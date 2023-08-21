package src

import (
	"net"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

const defaultAnalysisDur = time.Minute

var _logger = logrus.WithField("comp", "mngr")

type Statistic struct {
	read, written int64
}

func (stat *Statistic) AddRead(n int64) {
	atomic.AddInt64(&stat.read, n)
}

func (stat *Statistic) AddWritten(n int64) {
	atomic.AddInt64(&stat.written, n)
}

type ConnectionMngr struct {
	// statistic
	stat   *Statistic
	active int32
}

func NewConnectionMngr() *ConnectionMngr {
	mngr := &ConnectionMngr{
		stat: &Statistic{},
	}
	go mngr.daemon()
	return mngr
}

func (mngr *ConnectionMngr) daemon() {
	timer := time.NewTimer(defaultAnalysisDur)
	for {
		<-timer.C
		_logger.Infof("[statistic] active=%d, read=%d, written=%d",
			atomic.LoadInt32(&mngr.active), atomic.LoadInt64(&mngr.stat.read), atomic.LoadInt64(&mngr.stat.written))
	}
}

func (mngr *ConnectionMngr) PipeHandler() TcpHandler {
	return TcpHandleFunc(func(ctx *Context) {
		p := NewPipe(ctx, mngr.stat)
		atomic.AddInt32(&mngr.active, 1)
		defer atomic.AddInt32(&mngr.active, -1)
		// do pipe
		p.Pipe()
	})
}

type Dialer interface {
	Dial(network, address string) (net.Conn, error)
}
