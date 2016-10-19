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
