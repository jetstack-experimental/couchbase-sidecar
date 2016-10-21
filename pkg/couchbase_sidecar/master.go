package couchbase_sidecar

import (
	"time"

	"github.com/Sirupsen/logrus"

	"github.com/dmaier-couchbase/cb-openshift3/tools/couchbase_sidecar/pkg/couchbase"
)

type master struct {
	cs     *CouchbaseSidecar
	cLocal *couchbase.Couchbase
}

func (m *master) Log() *logrus.Entry {
	return m.cs.Log().WithField("component", "master")
}

func (m *master) periodicCheck() error {

	if m.cLocal == nil {
		c, err := m.cs.CouchbaseLocal()
		if err != nil {
			return err
		}
		err = c.Connect()
		if err != nil {
			return err
		}
		m.cLocal = c
	}

	err := m.checkMemory()
	if err != nil {
		m.Log().Warnf("checking memory quota failed: %s", err)
	}

	return nil
}

func (m *master) checkRebalance() error {
	// TODO: Implement me
	return nil
}

func (m *master) checkMemory() error {
	ratio := 0.75
	dataQuota := int(float64(m.cs.couchbaseConfig.DataMemoryLimit) * ratio)
	indexQuota := int(float64(m.cs.couchbaseConfig.IndexMemoryLimit) * ratio)
	return m.cLocal.EnsureMemoryQuota(
		dataQuota,
		indexQuota,
	)
}

func (m *master) checkClusterID() error {
	// TODO: Implement me
	return nil
}

func (m *master) run() {
	m.Log().Info("starting")
	go func() {
		for {
			if m.cs.Master() {
				err := m.periodicCheck()
				if err != nil {
					m.Log().Warnf("Periodic master check failed: %s", err)
				}
			}
			time.Sleep(10 * time.Second)
		}
	}()
	<-m.cs.stopCh
	m.Log().Info("stopping")
}
