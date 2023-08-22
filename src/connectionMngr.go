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
	active int32
	quota  *QuotaMngr
	mngr   *ConnAccessMngr
}

func NewConnQuotaMngr() *ConnQuotaMngr {
	mngr := &ConnQuotaMngr{
		quota: &QuotaMngr{
			quotaWritten: defaultQuotaPerHour,
			quotaRead:    defaultQuotaPerHour,
		},
	}
	go mngr.daemon()
	return mngr
}

func (mngr *ConnQuotaMngr) daemon() {
	analysisT := time.NewTicker(defaultAnalysisDur)
	updateT := time.NewTicker(updateDur)
	for {
		select {
		case <-analysisT.C:
			_logger.Infof("[statistic] active=%d, read=%d, written=%d",
				atomic.LoadInt32(&mngr.active), atomic.LoadInt64(&mngr.quota.quotaRead), atomic.LoadInt64(&mngr.quota.quotaWritten))
		case <-updateT.C:
			mngr.quota.Update(defaultQuotaPerHour, defaultQuotaPerHour)
		}
	}
}

func (mngr *ConnQuotaMngr) PipeHandler() TcpHandler {
	return TcpHandleFunc(func(ctx *Context) {
		p := NewTcpPiper(ctx, mngr.quota)
		atomic.AddInt32(&mngr.active, 1)
		defer atomic.AddInt32(&mngr.active, -1)
		// do pipe
		p.Pipe()
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
		p := NewTcpPiper(ctx, nil)
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
