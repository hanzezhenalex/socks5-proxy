package src

import (
	"fmt"
	"net"

	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

const (
	handlerChainAbortIndex = 0xffff
	maxBufferSize          = 1 + 1 + 255 + 2
)

type Context struct {
	correlationId uuid.UUID
	Logger        *logrus.Entry
	from          net.Conn
	Pipe          *Pipe

	// for middleware
	handlers  []TcpHandler
	nextIndex int

	// for socks5 protocol
	Auth byte
	Cmd  byte
	Host string
	Port string
	buf  []byte
}

func NewContext(from net.Conn, handlers []TcpHandler) *Context {
	ctx := &Context{
		correlationId: uuid.NewV4(),
		from:          from,
		handlers:      handlers,
		nextIndex:     -1,
		buf:           make([]byte, maxBufferSize),
	}
	ctx.Logger = logrus.WithField("id", ctx.correlationId)
	ctx.Logger.Infof("new connection from %s", from.RemoteAddr().String())
	return ctx
}

func (c *Context) Close() {
	_ = c.from.Close()
	if c.Pipe != nil {
		_ = c.Pipe.Close()
	}
}

func (c *Context) Next() {
	c.nextIndex++

	for c.nextIndex < len(c.handlers) {
		h := c.handlers[c.nextIndex]
		h.ServeTcp(c)
		c.nextIndex++
	}
}

func (c *Context) Abort() {
	c.nextIndex = handlerChainAbortIndex
}

func (c *Context) AbortAndCloseSourceConn() {
	c.Abort()
	if err := c.SourceConn().Close(); err != nil {
		c.Logger.Warningf("fail to close source conn, err=%s", err.Error())
	}
}

func (c *Context) Buffer() []byte {
	return c.buf
}

func (c *Context) SetAuthMethod(m byte) {
	c.Auth = m
}

func (c *Context) TargetAddr() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}

func (c *Context) SourceConn() net.Conn {
	return c.from
}
