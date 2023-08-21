package src

import (
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

const defaultDialTimeout = time.Second * 30

type Pipe struct {
	ctx            *Context
	dialer         *net.Dialer
	source, target net.Conn

	read    int64
	written int64
}

func NewPipe(ctx *Context, dialer *net.Dialer) *Pipe {
	return &Pipe{
		ctx:    ctx,
		dialer: dialer,
		source: ctx.from,
	}
}

func (p *Pipe) Dial(addr string) (string, error) {
	if p.dialer == nil {
		p.dialer = &net.Dialer{
			Timeout: defaultDialTimeout,
		}
	}
	conn, err := p.dialer.Dial("tcp", addr)
	if err != nil {
		return "", err
	}
	p.target = conn
	return conn.LocalAddr().String(), nil
}

func (p *Pipe) Read(buf []byte) (int, error) {
	n, err := p.target.Read(buf)
	atomic.AddInt64(&p.read, int64(n))
	return n, err
}

func (p *Pipe) Write(buf []byte) (int, error) {
	n, err := p.target.Write(buf)
	atomic.AddInt64(&p.written, int64(n))
	return n, err
}

func (p *Pipe) Close() error {
	if p.target == nil {
		return nil
	}
	return p.target.Close()
}

func (p *Pipe) readLoop() {
	loop(p.source, p.target, p.ctx.Logger.WithField("loop", "read"))
}

func (p *Pipe) writeLoop() {
	loop(p.target, p.source, p.ctx.Logger.WithField("loop", "write"))
}

func loop(source, target net.Conn, logger *logrus.Entry) {
	_, err := io.Copy(source, target)
	if err != nil {
		logger.Errorf("loop err: %s", err.Error())
		if rErr, ok := err.(*net.OpError); ok {
			switch rErr.Op {
			case "read":
				if err := target.(*net.TCPConn).CloseWrite(); err != nil {
					logger.Warningf("fail to close(write) target conn, err=%s", err.Error())
				}
			case "write":
				if err := source.(*net.TCPConn).CloseRead(); err != nil {
					logger.Warningf("fail to close(read) source conn, err=%s", err.Error())
				}
			}
		}
		return
	}
	if err := source.(*net.TCPConn).CloseRead(); err != nil {
		logger.Warningf("fail to close(read) source conn, err=%s", err.Error())
	}
	if err := target.(*net.TCPConn).CloseWrite(); err != nil {
		logger.Warningf("fail to close(write) target conn, err=%s", err.Error())
	}
}

func (p *Pipe) Pipe() {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		p.readLoop()
	}()

	go func() {
		defer wg.Done()
		p.writeLoop()
	}()

	wg.Wait()
}
