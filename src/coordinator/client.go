package coordinator

import (
	"fmt"
	"io"
	"net/rpc"
)

type dialer func() (io.ReadWriteCloser, error)

type ClientCoordinator struct {
	srvAddr string
	dialer  dialer
}

func NewClientCoordinator(srvAddr string, dialer dialer) *ClientCoordinator {
	return &ClientCoordinator{
		srvAddr: srvAddr,
		dialer:  dialer,
	}
}

func (c *ClientCoordinator) FetchInstances() ([]InstanceInfo, error) {
	var resp FetchInstancesResp
	conn, err := c.dialer()
	if err != nil {
		return nil, fmt.Errorf("fail to dial to server, %w", err)
	}
	if err := rpc.NewClient(conn).Call("ServerCoordinator.FetchInstances", FetchInstancesParam{}, &resp); err != nil {
		return nil, fmt.Errorf("fail to fetch instances, %w", err)
	}
	return resp.Instances, nil
}
