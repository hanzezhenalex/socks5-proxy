package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/sirupsen/logrus"

	"socks5-proxy/src"
	"socks5-proxy/src/protocol"
)

var (
	agentIp    string
	agentPort  string
	serverIp   string
	serverPort string
)

func parse() {
	flag.StringVar(&agentIp, "agent-ip", "0.0.0.0", "socks agent ip")
	flag.StringVar(&agentPort, "agent-port", "1080", "socks agent port")
	flag.StringVar(&serverIp, "server-ip", "0.0.0.0", "socks server ip")
	flag.StringVar(&serverPort, "server-port", "1081", "socks server port")

	flag.Parse()
}

func main() {
	parse()

	serverAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%s", serverIp, serverPort))
	if err != nil {
		logrus.Errorf("fail to parse socks server ip/port, err=%s", err.Error())
		os.Exit(1)
	}

	agentAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%s", agentIp, agentPort))
	if err != nil {
		logrus.Errorf("fail to parse socks server ip/port, err=%s", err.Error())
		os.Exit(1)
	}

	s := src.NewTcpServer(agentAddr)
	s.Use(src.RecoveryHandler())

	s.Use(
		protocol.AuthMethodNegotiation([]byte{protocol.NoAuthenticationRequired}),
		protocol.Auth(),
		protocol.ClientSayHello(time.Second*30, serverAddr),
	)

	s.SetFinalHandler(src.Pipe())

	if err := s.ListenAndServe(); err != nil {
		logrus.Errorf("an error happened when serve tcp, err=%s", err.Error())
		os.Exit(1)
	}
}
