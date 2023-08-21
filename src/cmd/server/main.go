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
	ip    string
	port  string
	local bool
)

const defaultDialTimeout = time.Second * 30

func parse() {
	flag.StringVar(&ip, "ip", "0.0.0.0", "socks server ip")
	flag.StringVar(&port, "port", "1081", "socks server port")
	flag.BoolVar(&local, "local", false, "use local mode")

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
	mngr := src.NewConnectionMngr()

	s.Use(src.RecoveryHandler())
	registerMiddlewares(s, local)
	s.SetFinalHandler(mngr.PipeHandler())

	if err := s.ListenAndServe(); err != nil {
		logrus.Errorf("an error happened when serve tcp, err=%s", err.Error())
		os.Exit(1)
	}
}

func registerMiddlewares(s *src.TcpServer, local bool) {
	dialer := &net.Dialer{
		Timeout: defaultDialTimeout,
	}
	if local {
		logrus.Info("running in local mode")
		s.Use(
			protocol.AuthMethodNegotiation([]byte{protocol.NoAuthenticationRequired}),
			protocol.Auth(),
			protocol.CommandNegotiation([]byte{protocol.Connect}),
			protocol.Command(dialer),
		)
	} else {
		logrus.Info("running in remote mode")
		s.Use(
			protocol.ServerSayHello(),
			protocol.CommandNegotiation([]byte{protocol.Connect}),
			protocol.Command(dialer),
		)
	}
}
