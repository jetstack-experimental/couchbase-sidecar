package couchbase_sidecar

import (
	"github.com/dmaier-couchbase/cb-openshift3/tools/couchbase_sidecar/pkg/couchbase"
)

func (cs *CouchbaseSidecar) CouchbaseLocal() (*couchbase.Couchbase, error) {
	c, err := couchbase.New("http://127.0.0.1:8091")
	c.Username = cs.couchbaseConfig.Username
	c.Password = cs.couchbaseConfig.Password
	return c, err
}

func (cs *CouchbaseSidecar) CouchbaseLocalHealthy() error {
	c, err := cs.CouchbaseLocal()
	if err != nil {
		return err
	}
	return c.Healthy()
}

func (cs *CouchbaseSidecar) CouchbaseCluster() (*couchbase.Couchbase, error) {
	url, err := cs.couchbaseClusterURL()
	if err != nil {
		return nil, err
	}
	c, err := couchbase.New(url)
	c.Username = cs.couchbaseConfig.Username
	c.Password = cs.couchbaseConfig.Password
	return c, err
}

func (cs *CouchbaseSidecar) CouchbaseClusterHealthy() error {
	c, err := cs.CouchbaseCluster()
	if err != nil {
		return err
	}
	return c.Healthy()
}

func (cs *CouchbaseSidecar) JoinCluster() error {

	cLocal, err := cs.CouchbaseLocal()
	if err != nil {
		return err
	}

	cCluster, err := cs.CouchbaseCluster()
	if err != nil {
		return err
	}

	err = cCluster.Connect()
	if err != nil {
		return err
	}

	err = cLocal.Connect()
	if err == nil {
		cs.Log().Warnf("already initialized")
		return nil
	} else if err != couchbase.ErrorNodeUninitialized {
		return nil
	}

	err = cLocal.SetupAuth()
	if err != nil {
		return err
	}

	return cCluster.AddNode(cs.NodeName(), cs.couchbaseConfig.Username, cs.couchbaseConfig.Password, cs.couchbaseConfig.Services)
}

func (cs *CouchbaseSidecar) InitialiseCluster() error {

	c, err := cs.CouchbaseLocal()
	if err != nil {
		return err
	}

	err = c.Connect()

	if err == nil {
		return nil
	} else if err == couchbase.ErrorNodeUninitialized {
		err = c.UpdateMemoryDataQuota(cs.couchbaseConfig.DataMemoryLimit)
		if err != nil {
			return err
		}

		err = c.UpdateMemoryIndexQuota(cs.couchbaseConfig.IndexMemoryLimit)
		if err != nil {
			return err
		}

		err = c.UpdateServices(cs.couchbaseConfig.Services)
		if err != nil {
			return err
		}

		err = c.SetupAuth()
		if err != nil {
			return err
		}

	} else {
		return err
	}

	return nil
}
