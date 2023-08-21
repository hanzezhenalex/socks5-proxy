package protocol

import (
	"fmt"
	"io"
	"net"
	"strconv"

	"socks5-proxy/src"
)

const (
	version = 0x05
	rsv     = 0x00

	NoAuthenticationRequired = 0x00

	noAcceptMethods = 0xff

	ipv4   = 0x01
	domain = 0x03
	ipv6   = 0x04

	Connect = 0x01

	succeed                   = 0x00
	generalSocksServerFailure = 0x01
	networkUnreachable        = 0x03
	commandNotSupport         = 0x07
	addressTypeNotSupported   = 0x08
)

func checkVersion(v byte) error {
	if v == version {
		return nil
	}
	return fmt.Errorf("unknown protocol")
}

func AuthMethodNegotiation(allowedMethods []byte) src.TcpHandler {
	return src.TcpHandleFunc(func(ctx *src.Context) {
		conn := ctx.SourceConn()
		buf := ctx.Buffer()

		if _, err := io.ReadFull(conn, buf[:2]); err != nil {
			ctx.Logger.Warningf("fail to read message header from source conn, err=%s", err.Error())
			ctx.Abort()
			return
		}

		if err := checkVersion(buf[0]); err != nil {
			ctx.Logger.Warningf("fail to check version, err=%s", err.Error())
			ctx.AbortAndCloseSourceConn()
			return
		}

		n := int(buf[1])
		if _, err := io.ReadFull(conn, buf[:n]); err != nil {
			ctx.Logger.Warningf("fail to read auth methods to source conn, err=%s", err.Error())
			ctx.Abort()
			return
		}

		for _, allowed := range allowedMethods {
			for _, provided := range buf[:n] {
				if allowed == provided {
					if _, err := conn.Write([]byte{version, provided}); err != nil {
						ctx.Logger.Warningf("fail to send auth method to source conn, err=%s", err.Error())
						ctx.Abort()
					} else {
						ctx.Auth = provided
					}
					return
				}
			}
		}

		if _, err := conn.Write([]byte{version, noAcceptMethods}); err != nil {
			ctx.Logger.Warningf("fail to send noAcceptMethods to source conn, err=%s", err.Error())
			ctx.Abort()
		}
		ctx.Logger.Warningf("no accept methods")
	})
}

func Auth() src.TcpHandler {
	return src.TcpHandleFunc(func(ctx *src.Context) {
		switch ctx.Auth {
		case NoAuthenticationRequired:
			ctx.Logger.Info("no authentication required")
		default:
			ctx.Logger.Warningf("%x not implement yet", ctx.Auth)
			if err := ctx.SourceConn().Close(); err != nil {
				ctx.Logger.Warningf("fail to close source conn, err=%s", err.Error())
			}
			ctx.Abort()
		}
	})
}

func CommandNegotiation(allowedMethods []byte) src.TcpHandler {
	return src.TcpHandleFunc(func(ctx *src.Context) {
		conn := ctx.SourceConn()
		buf := ctx.Buffer()

		if _, err := io.ReadFull(conn, buf[:3]); err != nil {
			ctx.Logger.Errorf("fail to read header from source conn, err=%s", err.Error())
			ctx.Abort()
			return
		}
		if err := checkVersion(buf[0]); err != nil {
			ctx.Logger.Errorf("fail to check version, err=%s", err.Error())
			ctx.AbortAndCloseSourceConn()
			return
		}
		if continue_ := checkCommand(ctx, allowedMethods, buf); !continue_ {
			ctx.Logger.Warningf("command not support, command=%x", ctx.Cmd)
			if _, err := ctx.SourceConn().Write(commandErrorReply(commandNotSupport, buf)); err != nil {
				ctx.Logger.Errorf("fail to send commandNotSupport reply to source conn, err=%s", err.Error())
			}
			ctx.AbortAndCloseSourceConn()
		}

		continue_, err := readAddr(ctx, buf)
		if !continue_ {
			ctx.AbortAndCloseSourceConn()
		} else if err != nil {
			ctx.Logger.Errorf("fail to read addr from source conn, err=%s", err.Error())
			ctx.Abort()
		}
	})
}

func Command() src.TcpHandler {
	return src.TcpHandleFunc(func(ctx *src.Context) {
		conn := ctx.SourceConn()

		switch ctx.Cmd {
		case Connect:
			p := src.NewPipe(ctx, nil)
			addr, err := p.Dial(ctx.TargetAddr())
			if err != nil {
				ctx.Logger.Errorf("fail to connect to target conn, err=%s", err.Error())
				if _, err := conn.Write(commandErrorReply(networkUnreachable, ctx.Buffer())); err != nil {
					ctx.Logger.Warningf("fail to send reply, err=%s", err.Error())
					ctx.Abort()
					return
				}
				ctx.AbortAndCloseSourceConn()
				return
			}
			ctx.Pipe = p
			if _, err := conn.Write(commandSuccessReply(addr, ctx.Buffer())); err != nil {
				ctx.Logger.Errorf("fail to send command success reply, err=%s", err.Error())
				if err := p.Close(); err != nil {
					ctx.Logger.Warningf("fail to close pipe, err=%s", err.Error())
				}
				ctx.Abort()
			}
		default:
			ctx.Logger.Warningf("Cmd %x not implement yet", ctx.Cmd)
			if _, err := conn.Write(commandErrorReply(generalSocksServerFailure, ctx.Buffer())); err != nil {
				ctx.Logger.Warningf("fail to send reply, err=%s", err.Error())
				ctx.Abort()
				return
			}
			ctx.AbortAndCloseSourceConn()
		}
	})
}

func checkCommand(ctx *src.Context, allowedMethods []byte, buf []byte) bool {
	ctx.Cmd = buf[1]
	matched := false
	for _, allowed := range allowedMethods {
		if allowed == ctx.Cmd {
			matched = true
		}
	}
	return matched
}

func readAddr(c *src.Context, buf []byte) (bool, error) {
	conn := c.SourceConn()

	if _, err := io.ReadFull(conn, buf[:1]); err != nil {
		return false, err
	}

	switch buf[0] {
	case ipv4:
		if _, err := io.ReadFull(conn, buf[:net.IPv4len]); err != nil {
			return false, err
		}
		c.Host = net.IP(buf[:net.IPv4len]).String()
	case domain:
		if _, err := io.ReadFull(conn, buf[:1]); err != nil {
			return false, err
		}
		length := int(buf[0])
		if _, err := io.ReadFull(conn, buf[:length]); err != nil {
			return false, err
		}
		c.Host = string(buf[:length])
	case ipv6:
		if _, err := io.ReadFull(conn, buf[:net.IPv6len]); err != nil {
			return false, err
		}
		c.Host = net.IP(buf[:net.IPv6len]).String()
	default:
		_, err := conn.Write(commandErrorReply(addressTypeNotSupported, buf))
		return false, err
	}

	if _, err := io.ReadFull(conn, buf[:2]); err != nil {
		return false, err
	}

	c.Port = strconv.Itoa(int(buf[0])<<8 | int(buf[1]))
	return true, nil
}

// make sure parse the legal addr
func parseAddr(s string, buf []byte) []byte {
	host, port, _ := net.SplitHostPort(s)

	ip := net.ParseIP(host)
	if ip == nil {
		// domain name
		buf = append(buf, domain, byte(len(ip)))
		buf = append(buf, []byte(host)...)
	} else {
		if ip4 := ip.To4(); ip4 != nil {
			buf = append(buf, ipv4)
			buf = append(buf, ip4...)
		} else {
			buf = append(buf, ipv6)
			buf = append(buf, ip...)
		}
	}

	portn, _ := strconv.ParseUint(port, 10, 16)
	buf = append(buf, byte(portn>>8), byte(portn))
	return buf
}

func commandErrorReply(rep byte, buf []byte) []byte {
	ret := buf[:10]
	ret[0] = version
	ret[1] = rep
	for i := 2; i < len(ret); i++ {
		ret[i] = 0
	}
	return ret
}

func commandSuccessReply(addr string, buf []byte) []byte {
	buf = buf[:0]
	buf = append(buf, version, succeed, rsv)
	return parseAddr(addr, buf)
}
