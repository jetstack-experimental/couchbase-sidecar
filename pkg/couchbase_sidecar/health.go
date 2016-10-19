package couchbase_sidecar

import (
	"github.com/Sirupsen/logrus"
)

type healthCheck struct {
	cs *CouchbaseSidecar
}

func (m *healthCheck) Log() *logrus.Entry {
	return m.cs.Log().WithField("component", "healthCheck")
}

func (m *healthCheck) run() {
	m.Log().Info("starting")
	<-m.cs.stopCh
	m.Log().Info("stopping")
}
