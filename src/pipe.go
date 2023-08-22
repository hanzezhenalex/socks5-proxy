package src

import (
	"io"
	"net"
	"sync"

	"github.com/sirupsen/logrus"
)

type TcpConn interface {
	net.Conn
	CloseWrite() error
	CloseRead() error
}

type TcpPiper struct {
	ctx            *Context
	source, target TcpConn

	quota *QuotaMngr
}

func NewTcpPiper(ctx *Context, quota *QuotaMngr) *TcpPiper {
	p := &TcpPiper{
		ctx:    ctx,
		source: ctx.SourceConn().(*net.TCPConn),
		quota:  quota,
	}
	target := ctx.TargetConn().(*net.TCPConn)
	if quota != nil {
		p.target = quota.WrapTcpConnection(target)
	} else {
		p.target = target
	}
	return p
}

func (p *TcpPiper) Close() error {
	if p.target == nil {
		return nil
	}
	return p.target.Close()
}

func (p *TcpPiper) readLoop() {
	_, err := io.Copy(p.source, p.target)
	handleLoopError(err, p.source, p.target, p.ctx.Logger.WithField("loop", "read"))
}

func (p *TcpPiper) writeLoop() {
	_, err := io.Copy(p.target, p.source)
	handleLoopError(err, p.target, p.source, p.ctx.Logger.WithField("loop", "write"))
}

func handleLoopError(err error, source, target TcpConn, logger *logrus.Entry) {
	if err != nil {
		logger.Errorf("loop err: %s", err.Error())
		if rErr, ok := err.(*net.OpError); ok {
			switch rErr.Op {
			case "read":
				if err := target.CloseWrite(); err != nil {
					logger.Warningf("fail to close(write) target conn, err=%s", err.Error())
				}
			case "write":
				if err := source.CloseRead(); err != nil {
					logger.Warningf("fail to close(read) source conn, err=%s", err.Error())
				}
			}
		}
		return
	}
	if err := source.CloseRead(); err != nil {
		logger.Warningf("fail to close(read) source conn, err=%s", err.Error())
	}
	if err := target.CloseWrite(); err != nil {
		logger.Warningf("fail to close(write) target conn, err=%s", err.Error())
	}
}

func (p *TcpPiper) Pipe() {
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
