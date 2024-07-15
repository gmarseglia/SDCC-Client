#!/bin/bash

# clear
clear

if [ $# -lt 2 ]; then
        echo "Wrong usage ClientsNum MillisStep."
        exit
fi

# Set the number of iterations for the loop
num_iterations=$1
millis_step=$2
nano_step=$(($2 * 1000))

# print
echo "Launching $num_iterations clients, once every $millis_step ms."

# Define the command you want to run in the background
command_to_run="go run client.go"

# Loop through the specified number of iterations
for ((i=1; i<=$num_iterations; i++)); do
    # Run the command in the background using the '&' operator
    $command_to_run &

    usleep $nano_step
done



wait
