package coordinator

import (
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

var srvTracer = logrus.WithField("comp", "server")

type InstanceInfo struct {
	Addr string
}

type Instance struct {
	InstanceInfo
	lastRecv time.Time
}

func (ins *Instance) recvHeartBeat() {
	ins.lastRecv = time.Now()
}

func (ins *Instance) Ahead(deadline time.Time) bool {
	return ins.lastRecv.Before(deadline)
}

func NewInstance(addr string) *Instance {
	return &Instance{
		InstanceInfo: InstanceInfo{
			Addr: addr,
		},
		lastRecv: time.Now(),
	}
}

type ServerCoordinator struct {
	cleanInterval time.Duration
	noContactDur  time.Duration

	peers map[string]*Instance
	mutex sync.RWMutex // protect peers

	stopCh chan struct{}
}

func NewServerCoordinator() *ServerCoordinator {
	c := &ServerCoordinator{
		cleanInterval: 3 * time.Minute,
		noContactDur:  5 * time.Minute,
		peers:         make(map[string]*Instance),
		stopCh:        make(chan struct{}),
	}
	return c
}

func (c *ServerCoordinator) Daemon() {
	ticker := time.NewTicker(c.cleanInterval)
	srvTracer.Infof("coordinator start working, clean interval=%s", c.cleanInterval.String())

	for {
		select {
		case <-ticker.C:
			c.clean()
		case <-c.stopCh:
			return
		}
	}
}

func (c *ServerCoordinator) clean() {
	var toRemove []string
	var removed []*Instance
	deadline := time.Now().Add(-1 * c.noContactDur)

	c.mutex.RLock()
	for addr, p := range c.peers {
		if p.Ahead(deadline) {
			toRemove = append(toRemove, addr)
		}
	}
	c.mutex.RUnlock()

	if len(toRemove) > 0 {
		c.mutex.Lock()
		for _, addr := range toRemove {
			if p, ok := c.peers[addr]; ok && p.Ahead(deadline) {
				removed = append(removed, p)
				delete(c.peers, addr)
			}
		}
		c.mutex.Unlock()
	}
	if len(toRemove) > 0 {
		srvTracer.Infof("following instances are cleaned: %s", strings.Join(toRemove, ", "))
	}
}

func (c *ServerCoordinator) Stop() {
	close(c.stopCh)
}

/*
	RPC
*/

type HeartBeatParam struct {
	InstanceAddr string
}

type HeartBeatResp struct{}

func (c *ServerCoordinator) HeartBeat(param HeartBeatParam, _ *HeartBeatResp) error {
	c.mutex.RLock()
	p, ok := c.peers[param.InstanceAddr]
	if ok {
		p.recvHeartBeat()
		c.mutex.RUnlock()
	} else {
		c.mutex.RUnlock()

		srvTracer.Infof("new instances joined, addr=%s", param.InstanceAddr)

		c.mutex.Lock()
		defer c.mutex.Unlock()

		if p, ok := c.peers[param.InstanceAddr]; ok {
			p.recvHeartBeat()
		} else {
			c.peers[param.InstanceAddr] = NewInstance(param.InstanceAddr)
		}
	}
	return nil
}

type FetchInstancesParam struct{}

type FetchInstancesResp struct {
	Instances []InstanceInfo
}

func (c *ServerCoordinator) FetchInstances(_ FetchInstancesParam, resp *FetchInstancesResp) error {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	ret := make([]InstanceInfo, 0, len(c.peers))

	for _, ins := range c.peers {
		ret = append(ret, InstanceInfo{
			Addr: ins.Addr,
		})
	}
	resp.Instances = ret
	return nil
}
