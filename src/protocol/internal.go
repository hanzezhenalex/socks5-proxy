package protocol

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"time"

	"socks5-proxy/src"
)

var clientSecretKey = []byte("dfb06f") // hard code for now
var serverSecretKey = []byte("be6048")

func ClientSayHello(timeout time.Duration, addr net.Addr) src.TcpHandler {
	return src.TcpHandleFunc(func(ctx *src.Context) {
		target, err := net.DialTimeout("tcp", addr.String(), timeout)
		if err != nil {
			ctx.Error(err)
			return
		}

		ctx.Logger.Info("client say hello")
		if _, err := target.Write(clientSecretKey); err != nil {
			ctx.Error(err)
			return
		}

		buf := ctx.Buffer()
		buf = buf[:len(serverSecretKey)]

		ctx.Logger.Info("waiting for server")
		if _, err := io.ReadFull(target, buf); err != nil {
			ctx.Error(err)
			return
		}

		if bytes.Equal(buf, serverSecretKey) {
			ctx.Logger.Info("handshake successfully")
			ctx.To = target
			ctx.Host = target.RemoteAddr().String()
		} else {
			ctx.Error(fmt.Errorf("secret key mismatch, key=%s", string(buf)))
		}
	})
}

func ServerSayHello() src.TcpHandler {
	return src.TcpHandleFunc(func(ctx *src.Context) {
		conn := ctx.SourceConn()
		buf := ctx.Buffer()
		buf = buf[:len(clientSecretKey)]

		ctx.Logger.Info("waiting for client")
		if _, err := io.ReadFull(conn, buf); err != nil {
			ctx.Error(err)
			return
		}

		if !bytes.Equal(buf, clientSecretKey) {
			ctx.Error(fmt.Errorf("secret key mismatch, key=%s", string(buf)))
			return
		}

		ctx.Logger.Info("server say hello")
		if _, err := conn.Write(serverSecretKey); err != nil {
			ctx.Error(err)
			return
		}

		ctx.Logger.Info("handshake successfully")
	})
}
