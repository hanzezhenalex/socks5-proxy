#!/bin/bash

set -euxo pipefail

# test 1: local mode

source ../scripts/deploy.sh

curl http://wwww.baidu.com -x socks5h://127.0.0.1:1081

docker container stop "${SEVER_DOCKER_ID}"

