#!/bin/bash

source ./docker/names.config

if [[ $# -eq 1 ]]; then

    if [[ $1 -eq "-i" ]]; then
        docker run --rm -it --name=$CLIENT_CONTAINER_NAME $CLIENT_IMAGE_NAME:latest
    fi

    if [[ $1 -eq "-e" ]]; then
        docker run --rm -d -e FrontAddr='127.0.0.1' --name=$CLIENT_CONTAINER_NAME $CLIENT_IMAGE_NAME:latest
    fi

    if [[ $1 -eq "-n" ]]; then
        docker run --rm -it --network sdcc-final_net sdcc-client:latest
    fi

    if [[ $1 -eq "-3" ]]; then
        docker run --rm -it --network 3-worker_net sdcc-client:latest
    fi

else
    docker run --rm -d --name=$CLIENT_CONTAINER_NAME $CLIENT_IMAGE_NAME:latest
fi

