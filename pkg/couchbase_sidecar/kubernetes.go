package couchbase_sidecar

import (
	"fmt"
	"strings"

	kube "k8s.io/client-go/1.5/kubernetes"
	kubeAPI "k8s.io/client-go/1.5/pkg/api/v1"
	kubeREST "k8s.io/client-go/1.5/rest"
)

func (cs *CouchbaseSidecar) KubernetesClientset() *kube.Clientset {
	if cs.kubernetesClientset == nil {
		config, err := kubeREST.InClusterConfig()
		if err != nil {
			cs.Log().Fatalf("Error creating kubernetes in cluster config: %s", err)
		}
		// creates the clientset
		cs.kubernetesClientset, err = kube.NewForConfig(config)
		if err != nil {
			cs.Log().Fatalf("Error creating kubernetes clientset: %s", err)
		}
	}

	return cs.kubernetesClientset
}

func (cs *CouchbaseSidecar) Pod() *kubeAPI.Pod {
	if cs.pod == nil {
		pod, err := cs.KubernetesClientset().Core().Pods(cs.PodNamespace).Get(cs.PodName)
		if err != nil {
			cs.Log().Fatalf("Cannot find my own pod: %s", err)
		}
		cs.pod = pod
	}
	return cs.pod
}

func (cs *CouchbaseSidecar) Master() bool {
	// TODO: Master election, currently master hardcoded to first pod in data petset
	return fmt.Sprintf("%s-data-0", cs.couchbaseClusterName) == cs.PodName
}

func (cs *CouchbaseSidecar) readLabels() error {

	// read couchbase services

	servicesMap := map[string]string{
		"index": "index",
		"data":  "data",
		"query": "query",
	}

	services := []string{}

	types, ok := cs.Pod().Labels["type"]
	if !ok {
		return fmt.Errorf("pod label 'type' is not specifying the services of this node")
	}

	for _, service := range strings.Split(strings.ToLower(types), ",") {
		name, ok := servicesMap[service]
		if ok {
			services = append(services, name)
		}
	}

	if len(services) == 0 {
		return fmt.Errorf("pod label 'type' is not specifying a single valid service")
	}
	cs.couchbaseServices = services

	// read couchbase cluster name
	clusterName, ok := cs.Pod().Labels["name"]
	if !ok {
		return fmt.Errorf("pod label 'name' is not specifying the cluster name")
	}
	cs.couchbaseClusterName = strings.ToLower(clusterName)

	return nil
}
