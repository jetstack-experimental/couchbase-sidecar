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
	a.log().Debugf("received '%s' hook", name)
	if name == "stop" {
		return a.cs.StopHook()
	}

	return fmt.Errorf("Unknown hook name '%s'", name)
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

func (cs *CouchbaseSidecar) RemoveMyself() error {
	cLocal, err := cs.CouchbaseLocal()
	if err != nil {
		return err
	}

	localNode, err := cLocal.Info()
	if err != nil {
		return err
	}

	cCluster, err := cs.CouchbaseCluster()
	if err != nil {
		return err
	}

	return cCluster.RemoveNodes([]string{localNode.Hostname})
}

func (cs *CouchbaseSidecar) StopHook() error {

	// make sure stop hook has been received twice
	cs.waitGroupStopHookReceived.Done()
	cs.waitGroupStopHookReceived.Wait()

	// stop all other threads
	cs.waitGroupStopHookOnce.Do(func() {
		cs.Log().Infof("received stop hook, shutdown worker routines")
		cs.waitGroupStopHookFinished.Add(1)
		go func() {
			defer cs.waitGroupStopHookFinished.Done()
			cs.Stop()
			cs.waitGroupWorkers.Wait()

			for {
				err := cs.RemoveMyself()
				if err == nil {
					break
				}
				cs.Log().Warnf("Removing node failed: %s", err)
				time.Sleep(5 * time.Second)
			}

			cs.Log().Infof("goodbye from the hook")
		}()
	})

	cs.waitGroupStopHookFinished.Wait()
	return nil
}
