package src

import (
	"io"
	"sync"
)

type TcpHandler interface {
	ServeTcp(ctx *Context)
}

type TcpHandleFunc func(ctx *Context)

func (fn TcpHandleFunc) ServeTcp(ctx *Context) {
	fn(ctx)
}

func RecoveryHandler() TcpHandler {
	return TcpHandleFunc(func(ctx *Context) {
		defer func() {
			if r := recover(); r != nil {
				ctx.Logger.Errorf("recovery from %s", r)
			}
			ctx.Close()
		}()
		ctx.Next()
	})
}

func Pipe() TcpHandler {
	return TcpHandleFunc(func(ctx *Context) {
		ctx.Logger.Infof("start piping, target addr=%s", ctx.TargetAddr())

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			_, _ = io.Copy(ctx.from, ctx.To)
			ctx.Close()
			wg.Done()
		}()

		go func() {
			_, _ = io.Copy(ctx.To, ctx.from)
			ctx.Close()
			wg.Done()
		}()

		wg.Wait()
		ctx.Logger.Infof("finish piping")
	})
}
