#!/bin/bash

set -x

make docker_proxy

docker run -d --rm --network host alex/socks5-server:latest -- --local