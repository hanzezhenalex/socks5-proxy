package protocol

import (
	"bytes"
	"io"
	"net"

	"socks5-proxy/src"
)

var clientSecretKey = []byte("dfb06f") // hard code for now
var serverSecretKey = []byte("be6048")

func ClientSayHello(dialer src.Dialer, addr net.Addr) src.TcpHandler {
	return src.TcpHandleFunc(func(ctx *src.Context) {
		conn, err := dialer.Dial("tcp", addr.String())
		if err != nil {
			ctx.Logger.Errorf("fail to connect to target conn, err=%s", err.Error())
			ctx.AbortAndCloseSourceConn()
			return
		}

		ctx.Logger.Info("client say hello")
		if _, err := conn.Write(clientSecretKey); err != nil {
			ctx.Logger.Errorf("fail to say hello, err=%s", err.Error())
			ctx.AbortAndCloseSourceConn()
			return
		}

		buf := ctx.Buffer()
		buf = buf[:len(serverSecretKey)]

		ctx.Logger.Info("waiting for server")
		if _, err := io.ReadFull(conn, buf); err != nil {
			ctx.Logger.Errorf("fail to recieve hello, err=%s", err.Error())
			ctx.AbortAndCloseSourceConn()
			return
		}

		if bytes.Equal(buf, serverSecretKey) {
			ctx.Logger.Info("handshake successfully")
			ctx.SetTargetConn(conn)
			ctx.Host = conn.RemoteAddr().String()
		} else {
			ctx.Logger.Errorf("secret key mismatch, key=%s", string(buf))
			ctx.AbortAndCloseSourceConn()
		}
	})
}

func ServerSayHello() src.TcpHandler {
	return src.TcpHandleFunc(func(ctx *src.Context) {
		conn := ctx.SourceConn()
		buf := ctx.Buffer()
		buf = buf[:len(clientSecretKey)]

		ctx.Logger.Debug("waiting for client")
		if _, err := io.ReadFull(conn, buf); err != nil {
			ctx.Logger.Errorf("fail to recieve hello, err=%s", err.Error())
			ctx.Abort()
			return
		}

		if !bytes.Equal(buf, clientSecretKey) {
			ctx.Logger.Errorf("secret key mismatch, key=%s", string(buf))
			ctx.AbortAndCloseSourceConn()
			return
		}

		ctx.Logger.Debug("server say hello")
		if _, err := conn.Write(serverSecretKey); err != nil {
			ctx.Logger.Errorf("fail to send hello, err=%s", err.Error())
			ctx.Abort()
			return
		}

		ctx.Logger.Info("handshake successfully")
	})
}
