package src

import (
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
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
	_, err := io.Copy(p.source, p)
	if err != nil {
		p.ctx.Logger.Errorf("read loop err: %s", err.Error())
		if rErr, ok := err.(*net.OpError); ok && rErr.Op == "read" {
			if err := p.target.Close(); err != nil {
				p.ctx.Logger.Warningf("fail to close source conn, err=%s", err.Error())
			}
		}
	}

	if err := p.source.Close(); err != nil {
		p.ctx.Logger.Warningf("fail to close target conn, err=%s", err.Error())
	}
}

func (p *Pipe) writeLoop() {
	_, err := io.Copy(p, p.source)
	if err != nil {
		p.ctx.Logger.Errorf("read loop err: %s", err.Error())
		if rErr, ok := err.(*net.OpError); ok && rErr.Op == "read" {
			if err := p.source.Close(); err != nil {
				p.ctx.Logger.Warningf("fail to close source conn, err=%s", err.Error())
			}
		}
	}

	if err := p.target.Close(); err != nil {
		p.ctx.Logger.Warningf("fail to close target conn, err=%s", err.Error())
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
