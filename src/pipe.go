package src

import (
	"io"
	"net"
	"sync"

	"github.com/sirupsen/logrus"
)

type Pipe struct {
	ctx            *Context
	source, target net.Conn

	stat *Statistic
}

func NewPipe(ctx *Context, stat *Statistic) *Pipe {
	return &Pipe{
		ctx:    ctx,
		source: ctx.from,
		stat:   stat,
	}
}

func (p *Pipe) SetStatistic(stat *Statistic) {
	p.stat = stat
}

func (p *Pipe) Read(buf []byte) (int, error) {
	n, err := p.target.Read(buf)
	p.stat.AddRead(int64(n))
	return n, err
}

func (p *Pipe) Write(buf []byte) (int, error) {
	n, err := p.target.Write(buf)
	p.stat.AddWritten(int64(n))
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
	p.ctx.Logger.Infof("start piping, target addr=%s", p.ctx.TargetAddr())

	go func() {
		defer wg.Done()
		p.readLoop()
	}()

	go func() {
		defer wg.Done()
		p.writeLoop()
	}()

	wg.Wait()
	p.ctx.Logger.Infof("finish piping")
}
