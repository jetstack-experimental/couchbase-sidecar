package couchbase_sidecar

import (
	"time"

	"github.com/Sirupsen/logrus"
)

type master struct {
	cs *CouchbaseSidecar
}

func (m *master) Log() *logrus.Entry {
	return m.cs.Log().WithField("component", "master")
}

func (m *master) run() {
	m.Log().Info("starting")
	go func() {
		for {
			if m.cs.Master() {
				m.Log().Info("I am master, doing master things")
				//err := m.cs.InitialiseCluster()
				//m.Log().Infof("Initalise cluster: %s", err)
			}
			time.Sleep(10 * time.Second)
		}
	}()
	<-m.cs.stopCh
	m.Log().Info("stopping")
}
