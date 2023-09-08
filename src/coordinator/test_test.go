package coordinator

import (
	"io"
	"net"
	"net/rpc"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

const (
	serverAddr = "localhost:9000"

	heartBeatInterval = 10 * time.Millisecond
	cleanInterval     = 100 * time.Millisecond
	timeoutHeartBeat  = 100 * time.Millisecond
)

var testTracer = logrus.WithField("comp", "test")

func createRpcServer() (*ServerCoordinator, *rpc.Server, error) {
	server := NewServerCoordinator()
	rpcSrv := rpc.NewServer()
	if err := rpcSrv.Register(server); err != nil {
		return nil, nil, err
	}
	go server.Daemon()
	return server, rpcSrv, nil
}

func TestHearBeat(t *testing.T) {
	rq := require.New(t)

	var wg sync.WaitGroup
	wg.Add(1)

	_, srv, err := createRpcServer()
	rq.NoError(err)
	go func() {
		listener, err := net.Listen("tcp4", serverAddr)

		rq.NoError(err)
		conn, err := listener.Accept()
		rq.NoError(err)
		srv.ServeConn(conn)
		_ = listener.Close()
		wg.Done()
	}()

	ins1 := NewInstanceCoordinator(func() (io.ReadWriteCloser, error) {
		return net.Dial("tcp4", serverAddr)
	}, "", "")
	rq.NoError(ins1.heartBeat())

	wg.Wait()
}

func TestHeartBeat_Loop(t *testing.T) {
	rq := require.New(t)
	dialer := func() (io.ReadWriteCloser, error) {
		return net.Dial("tcp4", serverAddr)
	}

	var instances []*InstanceCoordinator
	var listener net.Listener

	c, srv, err := createRpcServer()
	rq.NoError(err)

	go func() {
		c.cleanInterval = cleanInterval
		c.noContactDur = timeoutHeartBeat
		l, err := net.Listen("tcp4", serverAddr)
		listener = l
		rq.NoError(err)
		srv.Accept(listener)
	}()

	for i := 0; i < 3; i++ {
		ins := NewInstanceCoordinator(dialer, "", strconv.FormatInt(int64(i), 10))
		ins.heartBeatInterval = heartBeatInterval
		go ins.Daemon()
		instances = append(instances, ins)
	}

	client := NewClientCoordinator(serverAddr, dialer)
	fetchInstances := func() []InstanceInfo {
		ret, err := client.FetchInstances()
		rq.NoError(err)
		return ret
	}

	// normal run
	time.Sleep(5 * cleanInterval)
	testTracer.Infof("normal run")
	rq.Equal(len(instances), len(fetchInstances()))

	// instances[0] disconnected
	instances[0].Stop()
	testTracer.Infof("instances[0] disconnected")

	time.Sleep(5 * cleanInterval)
	rq.Equal(len(instances)-1, len(fetchInstances()))

	// recover
	go instances[0].Daemon()
	testTracer.Infof("instances[0] go back")
	time.Sleep(5 * heartBeatInterval)
	rq.Equal(len(instances), len(fetchInstances()))

	_ = listener.Close()
	c.Stop()
	for _, ins := range instances {
		ins.Stop()
	}
}
