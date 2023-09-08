package coordinator

import (
	"fmt"
	"net/rpc"
	"time"

	"github.com/sirupsen/logrus"
)

var insTracer = logrus.WithField("comp", "instanceClient")

type InstanceCoordinator struct {
	dialer            dialer
	instanceAddr      string
	heartBeatInterval time.Duration

	stopCh chan struct{}
}

func NewInstanceCoordinator(dialer dialer, instanceIp, proxyPort string) *InstanceCoordinator {
	ins := &InstanceCoordinator{
		dialer:            dialer,
		instanceAddr:      fmt.Sprintf("%s:%s", instanceIp, proxyPort),
		heartBeatInterval: time.Minute,
	}
	return ins
}

func (c *InstanceCoordinator) Daemon() {
	c.stopCh = make(chan struct{})
	ticker := time.NewTicker(c.heartBeatInterval)
	insTracer.Info("client start working...")

LOOP:
	for {
		select {
		case <-ticker.C:
			if err := c.heartBeat(); err != nil {
				insTracer.Errorf("fail to send heart beat req to server, err=%s", err.Error())
			}
		case <-c.stopCh:
			break LOOP
		}
	}

	insTracer.Info("client stopped")
}

func (c *InstanceCoordinator) heartBeat() error {
	param := HeartBeatParam{
		InstanceAddr: c.instanceAddr,
	}
	var reply HeartBeatResp
	conn, err := c.dialer()
	defer func() {
		_ = conn.Close()
	}()
	if err != nil {
		return fmt.Errorf("fail to dial to server, %w", err)
	}
	if err := rpc.NewClient(conn).Call("ServerCoordinator.HeartBeat", param, &reply); err != nil {
		return fmt.Errorf("fail to call rpc HeartBeat, %w", err)
	}
	return nil
}

func (c *InstanceCoordinator) Stop() {
	close(c.stopCh)
}
