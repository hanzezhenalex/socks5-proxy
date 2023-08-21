package src

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

func PipeHandler() TcpHandler {
	return TcpHandleFunc(func(ctx *Context) {
		ctx.Logger.Infof("start piping, target addr=%s", ctx.TargetAddr())
		ctx.Pipe.Pipe()
		ctx.Logger.Infof("finish piping")
	})
}
