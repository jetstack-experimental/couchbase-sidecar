package couchbase_sidecar

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/couchbase/go-couchbase"
	"github.com/spf13/cobra"
	kube "k8s.io/client-go/1.5/kubernetes"
	kubeAPI "k8s.io/client-go/1.5/pkg/api/v1"
	kubeREST "k8s.io/client-go/1.5/rest"
)

var AppVersion string = "unknown"
var AppGitCommit string = "unknown"
var AppGitState string = "unknown"
var AppName string = "couchbase-sidecar"
var AppDesc string = "manage couchbase instance in kubernetes"

type CouchbaseSidecar struct {
	RootCmd             *cobra.Command
	kubernetesClientset *kube.Clientset
	Kubeconfig          string
	resyncPeriod        time.Duration

	// my pods representation
	pod          *kubeAPI.Pod
	PodName      string
	PodNamespace string

	// couchbase infors
	couchbaseURL      string
	couchbaseUsername string
	couchbasePassword string

	// stop channel for shutting down
	stopCh chan struct{}
}

func New() *CouchbaseSidecar {
	cs := &CouchbaseSidecar{
		resyncPeriod:      5 * time.Minute,
		stopCh:            make(chan struct{}),
		couchbaseURL:      "http://127.0.0.1:8091",
		couchbaseUsername: "admin",
		couchbasePassword: "jetstack",
	}
	cs.init()
	return cs
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
	client, err := couchbase.Connect(cs.couchbaseURL)
	if err != nil {
		return fmt.Errorf("Error connecting to local couchbase: %s", err)
	}
	log.Debugf("couchbase client=%+v", client)

	pool, err := client.GetPool("default")
	if err != nil {
		return fmt.Errorf("Error getting default pool: %s", err)
	}
	log.Debugf("couchbase pool=%+v", pool)

	return nil
}

func (cs *CouchbaseSidecar) init() {

	log.SetOutput(os.Stderr)
	log.SetLevel(log.DebugLevel)

	cs.RootCmd = &cobra.Command{
		Use:   AppName,
		Short: AppDesc,
		Run: func(cmd *cobra.Command, args []string) {
			cs.readEnvironmentVariables()

			log.Infof("Got pods info pod:=%#v", cs.Pod())

			for {
				err := cs.connectCouchbase()
				if err != nil {
					log.Warnf("Error connecting couchbase: %s", err)
				}
				time.Sleep(10 * time.Second)
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

func (cs *CouchbaseSidecar) readEnvironmentVariables() {

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
		log.Fatalf("Missing environment variable(s): %s", strings.Join(missingEnv, ", "))
	}
}

func (cs *CouchbaseSidecar) KubernetesClientset() *kube.Clientset {
	if cs.kubernetesClientset == nil {
		config, err := kubeREST.InClusterConfig()
		if err != nil {
			log.Fatalf("Error creating kubernetes in cluster config: %s", err)
		}
		// creates the clientset
		cs.kubernetesClientset, err = kube.NewForConfig(config)
		if err != nil {
			log.Fatalf("Error creating kubernetes clientset: %s", err)
		}
	}

	return cs.kubernetesClientset
}

func (cs *CouchbaseSidecar) Pod() *kubeAPI.Pod {
	if cs.pod == nil {
		pod, err := cs.KubernetesClientset().Core().Pods(cs.PodNamespace).Get(cs.PodName)
		if err != nil {
			log.Fatalf("Cannot find my own pod: %s", err)
		}
		cs.pod = pod
	}
	return cs.pod
}
