package src

import (
	"net"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	defaultAnalysisDur = 2 * time.Minute
	updateDur          = 24 * time.Hour
)

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
	analysisT := time.NewTicker(defaultAnalysisDur)
	updateT := time.NewTicker(updateDur)
	for {
		select {
		case <-analysisT.C:
			_logger.Infof("[statistic] active=%d, read=%d, written=%d",
				atomic.LoadInt32(&mngr.active), atomic.LoadInt64(&mngr.stat.read), atomic.LoadInt64(&mngr.stat.written))
		case <-updateT.C:
			_logger.Infof("[statistic] served read=%d, written=%d in last %f hours",
				atomic.LoadInt64(&mngr.stat.read), atomic.LoadInt64(&mngr.stat.written), updateDur.Hours())
			atomic.StoreInt64(&mngr.stat.read, 0)
			atomic.StoreInt64(&mngr.stat.written, 0)
		}
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
