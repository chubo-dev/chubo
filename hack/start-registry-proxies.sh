#!/usr/bin/env bash

# Sync changes with configuring-pull-through-cache.md.

set -e

factory_proxy_remoteurl="${CHUBO_IMAGE_FACTORY_URL:-${TALOS_IMAGE_FACTORY_URL:-https://factory.chubo.dev}}"
factory_proxy_name="${CHUBO_FACTORY_PROXY_NAME:-${TALOS_FACTORY_PROXY_NAME:-registry-factory.chubo.dev}}"

docker run -d -p 5000:5000 \
    -e REGISTRY_PROXY_REMOTEURL=https://registry-1.docker.io \
    --restart always \
    --name registry-docker.io registry:2

docker run -d -p 5003:5000 \
    -e REGISTRY_PROXY_REMOTEURL=https://gcr.io \
    --restart always \
    --name registry-gcr.io registry:2

docker run -d -p 5004:5000 \
    -e REGISTRY_PROXY_REMOTEURL=https://ghcr.io \
    --restart always \
    --name registry-ghcr.io registry:2

docker run -d -p 5006:5000 \
    -e REGISTRY_PROXY_REMOTEURL="${factory_proxy_remoteurl}" \
    --restart always \
    --name "${factory_proxy_name}" registry:2
