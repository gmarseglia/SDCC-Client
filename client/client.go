package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	pb "github.com/gmarseglia/SDCC-Common/proto"
	"github.com/gmarseglia/SDCC-Common/utils"
)

const (
	timeout    = 60 * time.Second
	msgMaxSize = 4 * 1024 * 1024
)

var (
	FrontAddr    = flag.String("FrontAddr", "", "The address to connect to.")
	FrontPort    = flag.String("FrontPort", "", "The port of the master service.")
	RequestCount = flag.String("RequestCount", "", "The number of requests to send.")
	Verbose      = flag.Bool("Verbose", false, "Enable verbose output.")
	TargetSize   = flag.Int("TargetSize", -1, "The target size of the image.")
	KernelNum    = flag.Int("KernelNum", -1, "The number of kernels.")
	KernelSize   = flag.Int("KernelSize", -1, "The size of the kernel.")
	AvgPoolSize  = flag.Int("AvgPoolSize", -1, "The size of the average pooling.")
	UseSigmoid   = flag.Bool("UseSigmoid", false, "Use sigmoid function.")
	RandomValues = flag.Bool("RandomValues", false, "Use random values.")
	ManualValues = flag.Bool("ManualValues", false, "Use manual values.")
	counter      int
	counterLock  sync.Mutex
	wg           sync.WaitGroup
	c            pb.FrontClient
)

func setupFields() {
	utils.SetupFieldMandatory(FrontAddr, "FrontAddr", func() {
		log.Printf("[Main]: FrontAddr field is mandatory.")
		exit()
	})
	utils.SetupFieldOptional(FrontPort, "FrontPort", "55555")
	utils.SetupFieldOptional(RequestCount, "RequestCount", "1")
	utils.SetupFieldBool(Verbose, "Verbose")
	utils.SetupFieldInt(false, TargetSize, "TargetSize", 500, nil)
	utils.SetupFieldInt(false, KernelNum, "KernelNum", 180, nil)
	utils.SetupFieldInt(false, KernelSize, "KernelSize", 3, nil)
	utils.SetupFieldInt(false, AvgPoolSize, "AvgPoolSize", 500, nil)
	utils.SetupFieldBool(UseSigmoid, "UseSigmoid")
	utils.SetupFieldBool(RandomValues, "RandomValues")
	utils.SetupFieldBool(ManualValues, "ManualValues")
}

func exit() {
	log.Printf("[Main]: All components stopped. Main component stopped. Goodbye.")
	os.Exit(0)
}

func convolutionalRun() {
	// Internal ID
	counterLock.Lock()
	counter++
	id := counter
	counterLock.Unlock()

	// Settings
	targetSize := *TargetSize
	kernelNum := *KernelNum
	kernelSize := *KernelSize
	avgPoolSize := *AvgPoolSize
	useKernels := kernelSize > 0
	useSigmoid := *UseSigmoid

	exptecedSize := max(
		(targetSize*targetSize*4)+(kernelSize*kernelSize*kernelNum)*4,
		targetSize*targetSize*kernelNum*4/(avgPoolSize*avgPoolSize))

	log.Printf("[Client]: Request #%d started. Target size: %d, Kernel size: %d, Kernel number: %d, Avg Pool Size: %d, Use Kernels: %v, Use Sigmoid: %v",
		id, targetSize, kernelSize, kernelNum, avgPoolSize, useKernels, useSigmoid)
	log.Printf("[Client]: Request #%d -> Expected size: %d, Expected results: %d", id, exptecedSize, kernelNum)

	if exptecedSize > msgMaxSize {
		log.Printf("[Client]: Request #%d NOT SENT -> Size must lower than: %d", id, msgMaxSize)
		wg.Done()
		return
	}

	// Produce the request
	frontRequest := &pb.ConvolutionalLayerFrontRequest{}

	// Set the target (input) matrix
	var target [][]float32
	if *ManualValues {
		target = utils.ManualInputMatrix("target", targetSize)
	} else {
		target = utils.GenerateMatrix(targetSize, targetSize, *RandomValues, 1)
	}
	frontRequest.Target = utils.MatrixToProto(target)

	// Set the kernels
	for i := 0; i < kernelNum; i++ {
		if *ManualValues {
			frontRequest.Kernel = append(frontRequest.Kernel, utils.MatrixToProto(utils.ManualInputMatrix(fmt.Sprintf("kernel %d", i), kernelSize)))
		} else {
			frontRequest.Kernel = append(frontRequest.Kernel, utils.MatrixToProto(utils.GenerateMatrix(kernelSize, kernelSize, *RandomValues, 1)))
		}
	}

	// Set the other fields
	frontRequest.AvgPoolSize = int32(avgPoolSize)
	frontRequest.UseKernels = useKernels
	frontRequest.UseSigmoid = useSigmoid

	// create the context
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// time the call
	startTime := time.Now()

	// contact the server
	r, err := c.ConvolutionalLayer(ctx, frontRequest)

	// check for errors
	if err != nil {
		if s, ok := status.FromError(err); ok {
			log.Printf("[Client]: Request #%d -> Unsuccessful! %s: %v", id, s.Message(), s.Details())
			wg.Done()
			return
		}
	}

	// print the result
	log.Printf("[Client]: Request #%d -> Response: (#%d) in %d ms, Results: %d",
		id,
		r.GetID(),
		time.Since(startTime).Milliseconds(),
		len(r.GetResult()))

	// print the result
	if *Verbose {
		utils.PrettyPrint("Target", target)
		for _, kernel := range frontRequest.Kernel {
			utils.PrettyPrint("Kernel", utils.ProtoToMatrix(kernel))
		}
		for _, result := range r.GetResult() {
			utils.PrettyPrint("Result", utils.ProtoToMatrix(result))
		}
	}

	wg.Done()
}

func main() {
	log.SetOutput(os.Stdout)

	// parse the flags
	flag.Parse()
	setupFields()

	// Welcome message
	requestCount, err := strconv.Atoi(*RequestCount)
	if err != nil {
		log.Printf("[Main]: RequestCount given is not a valid integer, reverting to default value: 1.")
		requestCount = 1
	}
	log.Printf("[Main]: Welcome. Client will send %d requests in parallel.", requestCount)

	// Set up a connection to the gRPC server
	serverFullAddr := fmt.Sprintf("%s:%s", *FrontAddr, *FrontPort)
	conn, err := grpc.Dial(serverFullAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("[Main]: Could not not connect. More:\n%v", err)
	}
	defer conn.Close()

	// create the client object
	c = pb.NewFrontClient(conn)

	for i := 0; i < requestCount; i++ {
		wg.Add(1)
		time.Sleep(time.Millisecond * time.Duration(100))
		go convolutionalRun()
	}

	// wait
	log.Printf("[Main]: All requests sent. Waiting for responses...")
	wg.Wait()

	log.Printf("[Main]: All requests completed. Terminating. Goodbye.")
}
