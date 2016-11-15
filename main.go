package main

import (
	"fmt"
	"github.com/dmaier-couchbase/cb-openshift3/tools/couchbase_sidecar/pkg/couchbase_sidecar"
	"os"
)

var (
	AppName      string = "couchbase-sidecar"
	AppDesc      string = "manage couchbase instance in kubernetes"
	AppVersion   string = "unknown"
	AppGitCommit string = "unknown"
	AppGitState  string = "unknown"
	AppBuildDate string = "unknown"
)

func main() {
	k := couchbase_sidecar.New()
	k.AppName = AppName
	k.AppDesc = AppDesc
	k.Version = AppVersion
	k.VersionDetail = fmt.Sprintf("git-commit=%s git-state=%s build-date=%s\n", AppGitCommit, AppGitState, AppBuildDate)

	if err := k.RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}
