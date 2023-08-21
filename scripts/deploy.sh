#!/bin/bash

set -x

make docker_server

docker run -d --rm --network host alex/socks5-server:latest --local