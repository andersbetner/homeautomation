#!/usr/bin/env bash
if [ -z "$MY_DOCKER_REGISTRY" ]; then
    echo "Must set env MY_DOCKER_REGISTRY=example.com:5000"
    exit 1
fi
version=1.4
env GOOS=linux GOARCH=arm GOARM=7 go build -o sense-arm
docker build -f Dockerfile.arm -t $MY_DOCKER_REGISTRY/arm-sense:$version .
docker push $MY_DOCKER_REGISTRY/arm-sense:$version
