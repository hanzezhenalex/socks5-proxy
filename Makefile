binaries: proxy

agent:
	go build -o $(GOPATH)/bin/s5agent ./src/cmd/agent/main.go

server:
	go build -o $(GOPATH)/bin/s5server ./src/cmd/server/main.go

# https://www.jetbrains.com/help/go/attach-to-running-go-processes-with-debugger.html
debug_agent:
	go build -gcflags="all=-N -l" -o $(GOPATH)/bin/s5agent ./src/cmd/agent/main.go
	dlv --listen=:2345 --headless=true --api-version=2 exec $(GOPATH)/bin/s5agent

debug_server:
	go build -gcflags="all=-N -l" -o $(GOPATH)/bin/s5server ./src/cmd/server/main.go
	dlv --listen=:2345 --headless=true --api-version=2 exec $(GOPATH)/bin/s5server

docker_proxy:
	docker build -f ./Dockerfile --target proxyServer -t alex/socks5-proxy .