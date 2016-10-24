package couchbase_sidecar

import (
	"reflect"
	"sort"
	"time"

	"github.com/Sirupsen/logrus"

	"github.com/dmaier-couchbase/cb-openshift3/tools/couchbase_sidecar/pkg/couchbase"
)

type master struct {
	cs                       *CouchbaseSidecar
	cLocal                   *couchbase.Couchbase
	nodesInactiveAdded       []string
	nodesInactiveAddedUpdate time.Time
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

	err = m.checkRebalance()
	if err != nil {
		m.Log().Warnf("checking for rebalance operation failed: %s", err)
	}

	return nil
}

func (m *master) checkRebalance() error {

	// TODO: check if a rebalance is currently in progress

	// check if nodes needs rebalancing
	nodes, err := m.cLocal.Nodes()
	if err != nil {
		return err
	}
	nodesInactiveAdded := []string{}
	nodesActive := []string{}
	for _, node := range nodes {
		if node.ClusterMembership == "inactiveAdded" {
			nodesInactiveAdded = append(nodesInactiveAdded, node.OTPNode)
		}
		if node.ClusterMembership == "active" {
			nodesActive = append(nodesActive, node.OTPNode)
		}
	}

	sort.Strings(nodesInactiveAdded)
	if !reflect.DeepEqual(nodesInactiveAdded, m.nodesInactiveAdded) {
		m.nodesInactiveAdded = nodesInactiveAdded
		m.nodesInactiveAddedUpdate = time.Now()
	}

	// no new node added for more than 30 secs
	if len(m.nodesInactiveAdded) > 0 && (time.Now().Sub(m.nodesInactiveAddedUpdate) > (time.Second * 30)) {
		knownNodes := []string{}
		knownNodes = append(knownNodes, nodesActive...)
		knownNodes = append(knownNodes, nodesInactiveAdded...)
		err := m.cLocal.Rebalance(knownNodes, []string{})
		if err != nil {
			return err
		}
		m.nodesInactiveAdded = []string{}
	}

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

	clusterID, err := m.cLocal.ClusterID()
	if err != nil {
		return err
	}

	return m.cs.UpdateClusterID(clusterID)
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
