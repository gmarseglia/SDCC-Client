FROM golang:1.21.6

WORKDIR /go/src/client

COPY /client .

RUN go build -v -o /usr/local/bin/client ./client.go

# CMD ["client"]