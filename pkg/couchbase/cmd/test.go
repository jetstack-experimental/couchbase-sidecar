package main

import (
	log "github.com/Sirupsen/logrus"

	"github.com/dmaier-couchbase/cb-openshift3/tools/couchbase_sidecar/pkg/couchbase"
)

func main() {
	log.SetLevel(log.DebugLevel)

	couchbase, err := couchbase.New("http://127.0.0.1:8091")
	couchbase.Username = "admin"
	couchbase.Password = "jetstack"
	if err != nil {
		log.Fatal("err: ", err)
	}
	log.Infof("couchbase:=%+v", couchbase)

	err = couchbase.Connect()
	if err != nil {
		log.Fatal("err: ", err)
	}

	err = couchbase.UpdateServices([]string{"data", "n1ql", "index"})
	if err != nil {
		log.Fatal("err: ", err)
	}

	err = couchbase.UpdateMemoryDataQuota(256)
	if err != nil {
		log.Fatal("err: ", err)
	}

	err = couchbase.UpdateMemoryIndexQuota(256)
	if err != nil {
		log.Fatal("err: ", err)
	}
}
