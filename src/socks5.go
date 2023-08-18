package src

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"time"
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

func AuthMethodNegotiation(allowedMethods []byte) TcpHandler {
	return TcpHandleFunc(func(ctx *Context) {
		var (
			conn = ctx.from
			buf  = ctx.Buffer()
		)

		if _, err := io.ReadFull(conn, buf[:2]); err != nil {
			ctx.Error(err)
			return
		}

		if err := checkVersion(buf[0]); err != nil {
			ctx.Error(err)
			return
		}

		n := int(buf[1])
		if _, err := io.ReadFull(conn, buf[:n]); err != nil {
			ctx.Error(err)
			return
		}

		for _, allowed := range allowedMethods {
			for _, provided := range buf[:n] {
				if allowed == provided {
					if _, err := conn.Write([]byte{version, provided}); err != nil {
						ctx.Error(err)
					} else {
						ctx.Auth = provided
					}
					return
				}
			}
		}

		if _, err := conn.Write([]byte{version, noAcceptMethods}); err != nil {
			ctx.Error(err)
		}
		ctx.logger.Warningf("no accept methods")
	})
}

func Auth() TcpHandler {
	return TcpHandleFunc(func(ctx *Context) {
		switch ctx.Auth {
		case NoAuthenticationRequired:
			ctx.logger.Info("no authentication required")
		default:
			ctx.logger.Warningf("%x not implement yet", ctx.Auth)
			if _, err := ctx.from.Write([]byte{version, noAcceptMethods}); err != nil {
				ctx.Error(err)
			}
		}
	})
}

func CommandNegotiation(allowedMethods []byte) TcpHandler {
	return TcpHandleFunc(func(ctx *Context) {
		var (
			conn = ctx.from
			buf  = ctx.Buffer()
		)

		if _, err := io.ReadFull(conn, buf[:3]); err != nil {
			ctx.Error(err)
			return
		}

		if err := checkVersion(buf[0]); err != nil {
			ctx.Error(err)
			return
		}

		if continue_, err := checkCommand(ctx, allowedMethods, buf); !continue_ || err != nil {
			ctx.Error(err)
			return
		}

		if continue_, err := readAddr(ctx, buf); !continue_ || err != nil {
			ctx.Error(err)
			return
		}
	})
}

func Command(timeout time.Duration) TcpHandler {
	return TcpHandleFunc(func(ctx *Context) {
		switch ctx.Cmd {
		case Connect:
			target, err := net.DialTimeout("tcp", ctx.TargetAddr(), timeout)
			if err != nil {
				if _, err := ctx.from.Write(commandErrorReply(networkUnreachable, ctx.Buffer())); err != nil {
					ctx.logger.Warningf("fail to send reply, err=%s", err.Error())
				}
				ctx.Error(err)
				return
			}
			ctx.To = target
			if _, err := ctx.from.Write(commandSuccessReply(ctx)); err != nil {
				ctx.Error(err)
			}
		default:
			ctx.logger.Warningf("Cmd %x not implement yet", ctx.Cmd)
			if _, err := ctx.from.Write(commandErrorReply(generalSocksServerFailure, ctx.Buffer())); err != nil {
				ctx.logger.Warningf("fail to send reply, err=%s", err.Error())
			}
			ctx.Abort()
		}
	})
}

func checkCommand(ctx *Context, allowedMethods []byte, buf []byte) (bool, error) {
	ctx.Cmd = buf[1]
	matched := false
	for _, allowed := range allowedMethods {
		if allowed == ctx.Cmd {
			matched = true
		}
	}

	if !matched {
		ctx.logger.Warningf("command not support, command=%x", ctx.Cmd)
		if _, err := ctx.from.Write(commandErrorReply(commandNotSupport, buf)); err != nil {
			return false, err
		}
	}
	return matched, nil
}

func readAddr(c *Context, buf []byte) (bool, error) {
	conn := c.from

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

func commandSuccessReply(c *Context) []byte {
	buf := c.Buffer()
	buf = buf[:0]
	buf = append(buf, version, succeed, rsv, ipv4)
	return parseAddr(c.To.LocalAddr().String(), buf)
}
