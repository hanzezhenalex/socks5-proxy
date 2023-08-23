#!/bin/bash

set -euxo pipefail

make docker_server

export SEVER_DOCKER_ID
SEVER_DOCKER_ID=$(docker run -d --rm --network host alex/socks5-server:latest --local)
echo "${SEVER_DOCKER_ID}"