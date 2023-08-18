package src

import (
	"fmt"
	"net"

	"github.com/sirupsen/logrus"
)

var logger = logrus.WithField("component", "tcp server")

type TcpServer struct {
	addr net.Addr

	handlers     []TcpHandler
	finalHandler TcpHandler
}

func NewTcpServer(addr net.Addr) *TcpServer {
	return &TcpServer{
		addr: addr,
	}
}

func (s *TcpServer) Use(handlers ...TcpHandler) {
	s.handlers = append(s.handlers, handlers...)
}

func (s *TcpServer) SetFinalHandler(handler TcpHandler) {
	s.finalHandler = handler
}

func (s *TcpServer) Handlers() []TcpHandler {
	ret := make([]TcpHandler, 0, len(s.handlers)+1)
	for _, h := range s.handlers {
		ret = append(ret, h)
	}
	ret = append(ret, s.finalHandler)
	return ret
}

func (s *TcpServer) ListenAndServe() error {
	listener, err := net.Listen("tcp", s.addr.String())
	if err != nil {
		return fmt.Errorf("fail to listen tcp socket, err=%w", err)
	}

	logger.Infof("start listen to tcp socket on %s", listener.Addr().String())

	for {
		conn, err := listener.Accept()
		if err != nil {
			return fmt.Errorf("fail to accept conn, err=%w", err)
		}

		go func() {
			ctx := NewContext(conn, s.Handlers())
			ctx.Next()
		}()
	}
}
