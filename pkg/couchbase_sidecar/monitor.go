package couchbase_sidecar

import (
	"fmt"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/dmaier-couchbase/cb-openshift3/tools/couchbase_sidecar/pkg/couchbase"
)

type monitor struct {
	cs *CouchbaseSidecar
}

func (m *monitor) Log() *logrus.Entry {
	return m.cs.Log().WithField("component", "monitor")
}

func (m *monitor) checkNode() error {

	// connect to node
	cLocal, err := m.cs.CouchbaseLocal()
	if err != nil {
		return err
	}
	err = cLocal.Connect()

	// initialize node if needed
	nodename := m.cs.NodeName()
	if err == couchbase.ErrorNodeUninitialized {
		m.Log().Infof("initializing node ...")
		err := cLocal.Initialize(nodename, m.cs.couchbaseConfig.Services)
		if err != nil {
			return fmt.Errorf("initializing node failed: %s", err)
		}
	} else if err != nil {
		return fmt.Errorf("connecting to node failed: %s", err)
	}

	// connect to cluster
	cCluster, err := m.cs.CouchbaseCluster()
	if err != nil {
		return fmt.Errorf("getting cluster failed: %s", err)
	}
	err = cCluster.Connect()
	if err != nil {
		return fmt.Errorf("connecting to cluster failed: %s", err)
	}

	// check if it needs to join cluster
	nodes, err := cCluster.Nodes()
	if err != nil {
		return fmt.Errorf("failed listing nodes in cluster: %s", err)
	}
	for _, node := range nodes {
		if node.Hostname == fmt.Sprintf("%s:8091", nodename) {
			return nil
		}
	}

	// join cluster
	err = cCluster.AddNode(
		nodename,
		m.cs.couchbaseConfig.Username,
		m.cs.couchbaseConfig.Password,
		m.cs.couchbaseConfig.Services,
	)

	return err

}

func (m *monitor) run() {
	m.Log().Info("starting")
	go func() {
		for {
			err := m.checkNode()
			if err != nil {
				m.Log().Warnf("problem checking node: %s", err)
			}
			time.Sleep(10 * time.Second)
		}
	}()
	<-m.cs.stopCh
	m.Log().Info("stopping")
}
