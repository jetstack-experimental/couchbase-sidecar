package couchbase_sidecar

import (
	"fmt"
	"io"
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
	PodIP        string
	serviceName  *string
	configMap    *kubeAPI.ConfigMap

	couchbaseConfig CouchbaseConfig

	// stop channel for shutting down
	stopCh chan struct{}

	// wait groups
	waitGroupWorkers sync.WaitGroup

	// graceful stop
	waitGroupStopHookReceived sync.WaitGroup
	waitGroupStopHookFinished sync.WaitGroup
	waitGroupStopHookOnce     sync.Once
}

func New() *CouchbaseSidecar {
	cs := &CouchbaseSidecar{
		resyncPeriod:              5 * time.Minute,
		stopCh:                    make(chan struct{}),
		waitGroupWorkers:          sync.WaitGroup{},
		waitGroupStopHookReceived: sync.WaitGroup{},
		waitGroupStopHookFinished: sync.WaitGroup{},
		couchbaseConfig: CouchbaseConfig{
			URL:      "http://127.0.0.1:8091",
			Username: "admin",
			Password: "jetstack",
		},
	}
	cs.waitGroupStopHookReceived.Add(2)
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
		Short: fmt.Sprintf("Print the version number of %s", AppName),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("%s version %s git-commit=%s git-state=%s\n", AppName, AppVersion, AppGitCommit, AppGitState)
		},
	}
	cs.RootCmd.AddCommand(versionCmd)

	stopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Asking sidecar to stop database application",
		Run: func(cmd *cobra.Command, args []string) {
			cs.Log().Infof("asking sidecar to stop")
			client, err := cs.RPCClient()
			if err != nil {
				cs.Log().Fatal("dialing:", err)
			}

			err = client.Call("App.Hook", "stop", nil)
			if err != nil {
				cs.Log().Fatal("sidecar stop error:", err)
			}
		},
	}
	cs.RootCmd.AddCommand(stopCmd)
}

func (cs *CouchbaseSidecar) copyMyself() error {

	destPath := "/sidecar"
	_, err := os.Stat(destPath)
	if err != nil {
		return err
	}

	sourcePath, err := os.Readlink("/proc/self/exe")
	if err != nil {
		return err
	}
	basename := filepath.Base(sourcePath)

	destPath = filepath.Join(destPath, basename)

	cs.Log().Debugf("copy myself from '%s' to '%s'", sourcePath, destPath)
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	sourceFileStat, err := sourceFile.Stat()
	if err != nil {
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		fmt.Errorf("%s is not a regular file", sourceFile)
	}

	destFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	return os.Chmod(destPath, 0755)

}

func (cs *CouchbaseSidecar) run() error {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		cs.Stop()
	}()

	// copy myself to folder
	cs.waitGroupWorkers.Add(1)
	go func() {
		defer cs.waitGroupWorkers.Done()
		err := cs.copyMyself()
		if err != nil {
			cs.Log().Warnf("Failed to provide binary to main container: %s", err)
		}
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

	cs.waitGroupWorkers.Wait()
	cs.waitGroupStopHookFinished.Wait()

	return nil
}

func (cs *CouchbaseSidecar) readEnvironmentVariables() error {

	cs.PodName = os.Getenv("POD_NAME")
	cs.PodNamespace = os.Getenv("POD_NAMESPACE")
	cs.PodIP = os.Getenv("POD_IP")

	missingEnv := []string{}

	if cs.PodName == "" {
		missingEnv = append(missingEnv, "POD_NAME")
	}

	if cs.PodNamespace == "" {
		missingEnv = append(missingEnv, "POD_NAMESPACE")
	}

	if cs.PodIP == "" {
		missingEnv = append(missingEnv, "POD_IP")
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
	cs.waitGroupWorkers.Add(1)
	go func() {
		defer cs.waitGroupWorkers.Done()
		cs.master.run()
	}()
}

func (cs *CouchbaseSidecar) startMonitor() {
	cs.monitor = &monitor{cs: cs}
	cs.waitGroupWorkers.Add(1)
	go func() {
		defer cs.waitGroupWorkers.Done()
		cs.monitor.run()
	}()
}

func (cs *CouchbaseSidecar) startHealthCheck() {
	cs.healthCheck = &healthCheck{cs: cs}
	cs.waitGroupWorkers.Add(1)
	go func() {
		defer cs.waitGroupWorkers.Done()
		cs.healthCheck.run()
	}()
}
