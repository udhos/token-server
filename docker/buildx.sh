#!/bin/bash

app=token-server

version=$(go run ./cmd/$app -version | awk '{ print $2 }' | awk -F= '{ print $2 }')

echo version=$version

#platforms=linux/amd64,linux/arm64
platforms=linux/amd64

#docker buildx create \
#    --use --platform=$platforms --name multi-platform-builder

docker buildx build \
   --no-cache \
   --push \
   --tag udhos/$app:latest \
   --tag udhos/$app:$version \
   --platform $platforms \
   -f ./docker/Dockerfile.buildx .

