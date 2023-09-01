package src

import (
	"net"
	"sync/atomic"
	"time"

	"github.com/alecthomas/units"
	"github.com/sirupsen/logrus"
)

const (
	defaultAnalysisDur  = 2 * time.Minute
	updateDur           = time.Hour
	defaultDialTimeout  = time.Second * 30
	defaultQuotaPerHour = int64(10 * units.GB)
)

var _logger = logrus.WithField("comp", "mngr")

type ConnMngr interface {
	Dialer() Dialer
	PipeHandler() TcpHandler
}

type ConnQuotaMngr struct {
	quota *QuotaMngr
	mngr  ConnMngr
}

func NewConnQuotaMngr() *ConnQuotaMngr {
	mngr := &ConnQuotaMngr{
		quota: &QuotaMngr{
			quotaWritten: defaultQuotaPerHour,
			quotaRead:    defaultQuotaPerHour,
		},
		mngr: NewConnAccessMngr(),
	}
	go mngr.daemon()
	return mngr
}

func (mngr *ConnQuotaMngr) daemon() {
	updateT := time.NewTicker(updateDur)
	for {
		<-updateT.C
		mngr.quota.Update(defaultQuotaPerHour, defaultQuotaPerHour)
	}
}

func (mngr *ConnQuotaMngr) PipeHandler() TcpHandler {
	fn := mngr.mngr.PipeHandler()
	return TcpHandleFunc(func(ctx *Context) {
		target, ok := ctx.TargetConn().(*net.TCPConn)
		if !ok {
			ctx.Logger.Error("target connection is not TCP connection.")
			ctx.Close()
			return
		}
		ctx.SetTargetConn(mngr.quota.WrapTcpConnection(target))

		fn.ServeTcp(ctx)
	})
}

func (mngr *ConnQuotaMngr) Dialer() Dialer {
	dialer := mngr.mngr.Dialer()
	return DialHandleFunc(func(network, address string) (net.Conn, error) {
		if !mngr.quota.Enough() {
			return nil, NotEnoughQuota
		}
		return dialer.Dial(network, address)
	})
}

type ConnAccessMngr struct {
	active int32
}

func NewConnAccessMngr() *ConnAccessMngr {
	mngr := &ConnAccessMngr{}
	go mngr.daemon()
	return mngr
}

func (mngr *ConnAccessMngr) PipeHandler() TcpHandler {
	return TcpHandleFunc(func(ctx *Context) {
		p, err := NewTcpPiper(ctx)
		if err != nil {
			ctx.Logger.Errorf("fail to create piper: err=%s", err.Error())
			ctx.Close()
			return
		}

		atomic.AddInt32(&mngr.active, 1)
		defer atomic.AddInt32(&mngr.active, -1)
		// do pipe
		p.Pipe()
	})
}

func (mngr *ConnAccessMngr) Dialer() Dialer {
	dialer := &net.Dialer{
		Timeout: defaultDialTimeout,
	}
	return DialHandleFunc(func(network, address string) (net.Conn, error) {
		return dialer.Dial(network, address)
	})
}

func (mngr *ConnAccessMngr) daemon() {
	analysisT := time.NewTicker(defaultAnalysisDur)
	for {
		<-analysisT.C
		_logger.Infof("[statistic] active=%d", atomic.LoadInt32(&mngr.active))
	}
}

type Dialer interface {
	Dial(network, address string) (net.Conn, error)
}

type DialHandleFunc func(network, address string) (net.Conn, error)

func (d DialHandleFunc) Dial(network, address string) (net.Conn, error) {
	return d(network, address)
}
