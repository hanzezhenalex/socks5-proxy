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
	logger        *logrus.Entry
	from          net.Conn

	// for middleware
	handlers  []TcpHandler
	nextIndex int

	// for socks5 protocol
	Auth byte
	Cmd  byte
	Host string
	Port string
	To   net.Conn
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
	ctx.logger = logrus.WithField("id", ctx.correlationId)
	ctx.logger.Infof("new connection from %s", from.RemoteAddr().String())
	return ctx
}

func (c *Context) Close() {
	_ = c.from.Close()
	if c.To != nil {
		_ = c.To.Close()
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

func (c *Context) Buffer() []byte {
	return c.buf
}

func (c *Context) SetAuthMethod(m byte) {
	c.Auth = m
}

func (c *Context) Error(err error) {
	if err != nil {
		c.logger.Error(err)
	}
	c.Abort()
}

func (c *Context) TargetAddr() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}
