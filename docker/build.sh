#!/bin/bash

source ./docker/names.config

echo $CLIENT_IMAGE_NAME

docker build -t $CLIENT_IMAGE_NAME -f $CLIENT_IMAGE_PATH .