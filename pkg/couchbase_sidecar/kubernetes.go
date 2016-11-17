package couchbase_sidecar

import (
	"fmt"
	"strings"

	kube "k8s.io/client-go/1.5/kubernetes"
	kubeResource "k8s.io/client-go/1.5/pkg/api/resource"
	kubeAPI "k8s.io/client-go/1.5/pkg/api/v1"
	kubeREST "k8s.io/client-go/1.5/rest"
)

const ConfigMapClusterID string = "couchbase.cluster-id"
const FailureDomainLabel string = "failure-domain.beta.kubernetes.io/zone"
const FailureDomainDefault string = "unknown"

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

func (cs *CouchbaseSidecar) ConfigMap() *kubeAPI.ConfigMap {
	if cs.configMap == nil {
		configMap, err := cs.KubernetesClientset().Core().ConfigMaps(cs.PodNamespace).Get(cs.couchbaseConfig.Name)
		if err != nil {
			cs.Log().Fatalf("Cannot find config map: %s", err)
		}
		cs.configMap = configMap
	}
	return cs.configMap
}

func (cs *CouchbaseSidecar) Master() bool {
	// TODO: Master election, currently master hardcoded to first pod in data petset
	// Has to be data node
	// TODO: Do that: http://k8s.io/kubernetes/pkg/client/leaderelection
	return fmt.Sprintf("%s-data-0", cs.couchbaseConfig.Name) == cs.PodName
}

func (cs *CouchbaseSidecar) readLabels() error {

	// read couchbase services

	servicesMap := map[string]string{
		"index": "index",
		"data":  "kv",
		"kv":    "kv",
		"query": "n1ql",
		"n1ql":  "n1ql",
	}

	services := []string{}

	types, ok := cs.Pod().Labels["type"]
	if !ok {
		return fmt.Errorf("pod label 'type' is not specifying the services of this node")
	}

	var mainType string

	for i, service := range strings.Split(strings.ToLower(types), ",") {
		name, ok := servicesMap[service]
		if ok {
			if i == 0 {
				mainType = service
			}
			services = append(services, name)
		}
	}

	if len(services) == 0 {
		return fmt.Errorf("pod label 'type' is not specifying a single valid service")
	}
	cs.couchbaseConfig.Services = services

	// read couchbase cluster name
	clusterName, ok := cs.Pod().Labels["name"]
	if !ok {
		return fmt.Errorf("pod label 'name' is not specifying the cluster name")
	}
	cs.couchbaseConfig.Name = strings.ToLower(clusterName)

	// TODO: detect the service name throught the PetSet variable
	serviceName := fmt.Sprintf("%s-%s", cs.couchbaseConfig.Name, mainType)
	cs.serviceName = &serviceName

	return nil
}

func (cs *CouchbaseSidecar) getMemoryLimitMi(input string) (int, error) {
	val, err := kubeResource.ParseQuantity(input)
	if err != nil {
		return 0, err
	}
	valInt := int(val.Value() / 1024 / 1024)
	if valInt < 256 {
		return 0, fmt.Errorf("minimum memory amount is 256Mi")
	}
	return valInt, err
}

func (cs *CouchbaseSidecar) readConfigMap() error {
	cm := cs.ConfigMap()
	var ok bool
	var err error

	if cs.couchbaseConfig.Username, ok = cm.Data["couchbase.username"]; !ok {
		return fmt.Errorf("Unable to read the username from ConfigMap")
	}

	if cs.couchbaseConfig.Password, ok = cm.Data["couchbase.password"]; !ok {
		return fmt.Errorf("Unable to read the password from ConfigMap")
	}

	key := "couchbase.index.memory-limit"
	if indexMemoryLimitStr, ok := cm.Data[key]; !ok {
		return fmt.Errorf("Unable to read '%s'", key)
	} else if cs.couchbaseConfig.IndexMemoryLimit, err = cs.getMemoryLimitMi(indexMemoryLimitStr); err != nil {
		return err
	}

	key = "couchbase.data.memory-limit"
	if dataMemoryLimitStr, ok := cm.Data[key]; !ok {
		return fmt.Errorf("Unable to read '%s'", key)
	} else if cs.couchbaseConfig.DataMemoryLimit, err = cs.getMemoryLimitMi(dataMemoryLimitStr); err != nil {
		return err
	}

	key = "couchbase.query.memory-limit"
	if queryMemoryLimitStr, ok := cm.Data[key]; !ok {
		return fmt.Errorf("Unable to read '%s'", key)
	} else if cs.couchbaseConfig.QueryMemoryLimit, err = cs.getMemoryLimitMi(queryMemoryLimitStr); err != nil {
		return err
	}

	// TODO: read bucket names / sample data
	return nil
}

func (cs *CouchbaseSidecar) couchbaseClusterURL() (string, error) {
	service, err := cs.KubernetesClientset().Core().Services(cs.PodNamespace).Get(cs.couchbaseConfig.Name)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("http://%s:8091", service.Spec.ClusterIP), nil
}

func (cs *CouchbaseSidecar) IsPetSet() (bool, error) {
	return true, nil
}

func (cs *CouchbaseSidecar) NodeName() string {
	if cs.serviceName == nil {
		return cs.PodIP
	}
	return fmt.Sprintf("%s.%s.%s", cs.PodName, *cs.serviceName, cs.DNSSuffix())
}

func (cs *CouchbaseSidecar) DNSSuffix() string {
	return fmt.Sprintf("%s.svc.cluster.local", cs.PodNamespace)
}

func (cs *CouchbaseSidecar) UpdateClusterID(clusterID string) error {
	client := cs.KubernetesClientset().Core().ConfigMaps(cs.PodNamespace)

	cm, err := client.Get(cs.couchbaseConfig.Name)
	if err != nil {
		return err
	}

	if val, ok := cm.Data[ConfigMapClusterID]; ok {
		if val == clusterID {
			return nil
		} else if val != "" {
			return fmt.Errorf("Cluster ID already set to: '%s'", val)
		}
	}

	cm.Data[ConfigMapClusterID] = clusterID
	_, err = client.Update(cm)
	return err
}

func (cs *CouchbaseSidecar) FailureDomain() string {
	nodeName := cs.Pod().Spec.NodeName

	node, err := cs.KubernetesClientset().Core().Nodes().Get(nodeName)
	if err != nil {
		cs.Log().Warnf("unable to get node object '%s': %s", nodeName, err)
		return FailureDomainDefault
	}

	if failureDomain, ok := node.GetLabels()[FailureDomainLabel]; ok {
		return failureDomain
	}

	cs.Log().Warnf("unable to find node label '%s'", FailureDomainLabel)
	return FailureDomainDefault
}
