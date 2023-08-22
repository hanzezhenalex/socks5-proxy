package src

import (
	"errors"
	"net"
	"sync/atomic"

	"github.com/alecthomas/units"
)

var NotEnoughQuota = errors.New("not enough quota")

const lowLevelMarker = int64(100 * units.KiB)

type QuotaMngr struct {
	quotaRead, quotaWritten int64
}

func (quota *QuotaMngr) Update(r, w int64) {
	atomic.StoreInt64(&quota.quotaRead, r)
	atomic.StoreInt64(&quota.quotaWritten, w)
}

func (quota *QuotaMngr) TryRead(n int64) bool {
	if left := atomic.AddInt64(&quota.quotaRead, -1*n); left < 0 {
		return false
	}
	return true
}

func (quota *QuotaMngr) TryWrite(n int64) bool {
	if left := atomic.AddInt64(&quota.quotaRead, -1*n); left < 0 {
		return false
	}
	return true
}

func (quota *QuotaMngr) Enough() bool {
	return atomic.LoadInt64(&quota.quotaRead)+atomic.LoadInt64(&quota.quotaWritten) > lowLevelMarker
}

func (quota *QuotaMngr) WrapTcpConnection(conn *net.TCPConn) *QuotaConn {
	return &QuotaConn{
		stat:    quota,
		TCPConn: conn,
	}
}

type QuotaConn struct {
	*net.TCPConn
	stat *QuotaMngr
}

func (c *QuotaConn) Read(buf []byte) (int, error) {
	n, err := c.TCPConn.Read(buf)
	if err != nil {
		return n, err
	}
	if ok := c.stat.TryRead(int64(n)); !ok {
		_ = c.Close()
		return n, &net.OpError{
			Op: "read", Net: c.RemoteAddr().Network(), Source: c.RemoteAddr(), Addr: c.LocalAddr(), Err: NotEnoughQuota,
		}
	}
	return n, err
}

func (c *QuotaConn) Write(buf []byte) (int, error) {
	n, err := c.TCPConn.Write(buf)
	if err != nil {
		return n, err
	}
	if ok := c.stat.TryWrite(int64(n)); !ok {
		_ = c.Close()
		return n, &net.OpError{
			Op: "write", Net: c.RemoteAddr().Network(), Source: c.RemoteAddr(), Addr: c.LocalAddr(), Err: NotEnoughQuota,
		}
	}
	return n, err
}
