package main

import (
	"fmt"
	"os"

	"github.com/dmaier-couchbase/cb-openshift3/tools/couchbase_sidecar/pkg/couchbase_sidecar"
)

func main() {
	k := couchbase_sidecar.New()
	if err := k.RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}
