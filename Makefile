binaries: proxy

proxy:
	go build -o $(GOPATH)/bin/s5proxy ./main.go

# https://www.jetbrains.com/help/go/attach-to-running-go-processes-with-debugger.html
debug_proxy:
	go build -gcflags="all=-N -l" -o $(GOPATH)/bin/s5proxy ./main.go
	dlv --listen=:2345 --headless=true --api-version=2 exec $(GOPATH)/bin/s5proxy

docker_proxy:
	docker build -f ./Dockerfile --target proxyServer -t alex/socks5-proxy .