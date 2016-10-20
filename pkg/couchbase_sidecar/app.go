package couchbase_sidecar

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/couchbase/go-couchbase"
	"github.com/spf13/cobra"
	kube "k8s.io/client-go/1.5/kubernetes"
	kubeAPI "k8s.io/client-go/1.5/pkg/api/v1"
)

var AppVersion string = "unknown"
var AppGitCommit string = "unknown"
var AppGitState string = "unknown"
var AppName string = "couchbase-sidecar"
var AppDesc string = "manage couchbase instance in kubernetes"

type CouchbaseConfig struct {
	URL              string
	Username         string
	Password         string
	QueryMemoryLimit int
	IndexMemoryLimit int
	DataMemoryLimit  int
	Name             string
	Services         []string
}

type CouchbaseSidecar struct {
	RootCmd             *cobra.Command
	kubernetesClientset *kube.Clientset
	Kubeconfig          string
	resyncPeriod        time.Duration

	// sub services
	master      *master
	healthCheck *healthCheck
	monitor     *monitor

	// my pods representation
	pod          *kubeAPI.Pod
	PodName      string
	PodNamespace string
	serviceName  *string
	configMap    *kubeAPI.ConfigMap

	couchbaseConfig CouchbaseConfig

	// stop channel for shutting down
	stopCh chan struct{}

	// wait group
	waitGroup sync.WaitGroup
}

func New() *CouchbaseSidecar {
	cs := &CouchbaseSidecar{
		resyncPeriod: 5 * time.Minute,
		stopCh:       make(chan struct{}),
		waitGroup:    sync.WaitGroup{},
		couchbaseConfig: CouchbaseConfig{
			URL:      "http://127.0.0.1:8091",
			Username: "admin",
			Password: "jetstack",
		},
	}
	cs.init()
	return cs
}

func (cs *CouchbaseSidecar) Log() *logrus.Entry {
	return logrus.WithField("context", "root")
}

func (cs *CouchbaseSidecar) userHomeDir() string {
	if runtime.GOOS == "windows" {
		home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return home
	}
	return os.Getenv("HOME")
}

func (cs *CouchbaseSidecar) connectCouchbase() error {
	client, err := couchbase.Connect(cs.couchbaseConfig.URL)
	if err != nil {
		return fmt.Errorf("Error connecting to local couchbase: %s", err)
	}
	cs.Log().Debugf("couchbase client=%+v", client)

	pool, err := client.GetPool("default")
	if err != nil {
		return fmt.Errorf("Error getting default pool: %s", err)
	}
	cs.Log().Debugf("couchbase pool=%+v", pool)

	return nil
}

func (cs *CouchbaseSidecar) init() {

	logrus.SetOutput(os.Stderr)
	logrus.SetLevel(logrus.DebugLevel)

	cs.RootCmd = &cobra.Command{
		Use:   AppName,
		Short: AppDesc,
		Run: func(cmd *cobra.Command, args []string) {

			err := cs.run()
			if err != nil {
				cs.Log().Fatalf("Error initializing side car", err)
			}
		},
	}
	cs.RootCmd.PersistentFlags().StringVarP(
		&cs.Kubeconfig,
		"kubeconfig",
		"k",
		filepath.Join(cs.userHomeDir(), ".kube/config"),
		"path to the kubeconfig file",
	)

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: fmt.Sprintf("Print the version number of %s", AppVersion),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("%s version %s git-commit=%s git-state=%s\n", AppName, AppVersion, AppGitCommit, AppGitState)
		},
	}

	cs.RootCmd.AddCommand(versionCmd)
}

func (cs *CouchbaseSidecar) run() error {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		cs.Stop()
	}()

	err := cs.readEnvironmentVariables()
	if err != nil {
		return err
	}

	err = cs.readLabels()
	if err != nil {
		return err
	}

	err = cs.readConfigMap()
	if err != nil {
		return err
	}

	cs.startMaster()
	cs.startMonitor()
	cs.startHealthCheck()

	cs.waitGroup.Wait()

	return nil
}

func (cs *CouchbaseSidecar) readEnvironmentVariables() error {

	cs.PodName = os.Getenv("POD_NAME")
	cs.PodNamespace = os.Getenv("POD_NAMESPACE")

	missingEnv := []string{}

	if cs.PodName == "" {
		missingEnv = append(missingEnv, "POD_NAME")
	}

	if cs.PodNamespace == "" {
		missingEnv = append(missingEnv, "POD_NAMESPACE")
	}

	if len(missingEnv) > 0 {
		return fmt.Errorf("Missing environment variable(s): %s", strings.Join(missingEnv, ", "))
	}

	return nil
}

func (cs *CouchbaseSidecar) Stop() {
	cs.Log().Info("shuting things down")
	close(cs.stopCh)
}

func (cs *CouchbaseSidecar) startMaster() {
	cs.Log().Infof("test")
	cs.master = &master{cs: cs}
	cs.waitGroup.Add(1)
	go func() {
		defer cs.waitGroup.Done()
		cs.master.run()
	}()
}

func (cs *CouchbaseSidecar) startMonitor() {
	cs.monitor = &monitor{cs: cs}
	cs.waitGroup.Add(1)
	go func() {
		defer cs.waitGroup.Done()
		cs.monitor.run()
	}()
}

func (cs *CouchbaseSidecar) startHealthCheck() {
	cs.healthCheck = &healthCheck{cs: cs}
	cs.waitGroup.Add(1)
	go func() {
		defer cs.waitGroup.Done()
		cs.healthCheck.run()
	}()
}
