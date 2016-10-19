package couchbase_sidecar

import (
	"github.com/Sirupsen/logrus"
)

type monitor struct {
	cs *CouchbaseSidecar
}

func (m *monitor) Log() *logrus.Entry {
	return m.cs.Log().WithField("component", "monitor")
}

func (m *monitor) run() {
	m.Log().Info("starting")
	<-m.cs.stopCh
	m.Log().Info("stopping")
}
