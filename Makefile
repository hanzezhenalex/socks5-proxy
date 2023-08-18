binaries: proxy

proxy:
	go build -o $(GOPATH)/bin/s5proxy ./main.go

docker_proxy:
	docker build -f ./Dockerfile --target proxyServer -t alex/socks5-proxy .