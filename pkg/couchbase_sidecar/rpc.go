package couchbase_sidecar

import (
	"fmt"
	"net/rpc"
	"time"

	"github.com/Sirupsen/logrus"
)

var AppRPCPath string = "/_rpc"

type AppRPC struct {
	cs *CouchbaseSidecar
}

func (a *AppRPC) log() *logrus.Entry {
	return a.cs.Log().WithField("component", "appRPC")
}

func (a *AppRPC) Hook(name string, result *bool) error {
	a.log().Infof("Received '%s' hook, waiting 10 secs", name)
	time.Sleep(10 * time.Second)
	a.log().Infof("Goodbye hook")
	return nil
}

func (m *healthCheck) newRPC() *rpc.Server {
	server := rpc.NewServer()
	app := &AppRPC{
		cs: m.cs,
	}
	server.RegisterName("App", app)
	return server
}

func (cs *CouchbaseSidecar) RPCClient() (*rpc.Client, error) {
	return rpc.DialHTTPPath("tcp", fmt.Sprintf("127.0.0.1:%d", AppListenPort), AppRPCPath)
}
