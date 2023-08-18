package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/sirupsen/logrus"

	"socks5-proxy/src"
)

var (
	ip   string
	port string
)

func parse() {
	flag.StringVar(&ip, "ip", "127.0.0.1", "socks server ip")
	flag.StringVar(&port, "port", "1080", "socks server port")

	flag.Parse()
}

func main() {
	parse()

	addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%s", ip, port))
	if err != nil {
		logrus.Errorf("fail to parse ip/port, err=%s", err.Error())
		os.Exit(1)
	}

	s := src.NewTcpServer(addr)
	s.Use(src.RecoveryHandler())
	s.Use(
		src.AuthMethodNegotiation([]byte{src.NoAuthenticationRequired}),
		src.Auth(),
		src.CommandNegotiation([]byte{src.Connect}),
		src.Command(time.Second*30),
	)
	s.SetFinalHandler(src.Pipe())

	if err := s.ListenAndServe(); err != nil {
		logrus.Errorf("an error happened when serve tcp, err=%s", err.Error())
		os.Exit(1)
	}
}
