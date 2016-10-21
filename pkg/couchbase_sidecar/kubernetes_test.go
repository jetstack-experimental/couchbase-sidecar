package couchbase_sidecar

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	kubeAPI "k8s.io/client-go/1.5/pkg/api/v1"
)

func NewSidecar(podName string) *CouchbaseSidecar {
	pod := &kubeAPI.Pod{}
	pod.Labels = exampleLabels()

	cm := &kubeAPI.ConfigMap{
		ObjectMeta: kubeAPI.ObjectMeta{
			Name:      "test",
			Namespace: "mynamespace",
		},
		Data: map[string]string{
			"couchbase.username":                   "user1",
			"couchbase.password":                   "password1",
			"sidecar.master":                       "podname-random",
			"sidecar.election-time":                "jstimestamp",
			"couchbase.query.memory-limit":         "2Gi",
			"couchbase.index.memory-limit":         "1024Mi",
			"couchbase.data.memory-limit":          "512Mi",
			"couchbase.bucket.${COUCHBASE_BUCKET}": "",
		},
	}

	return &CouchbaseSidecar{
		PodName:      podName,
		PodNamespace: "mynamespace",
		pod:          pod,
		configMap:    cm,
	}
}

func exampleLabels() map[string]string {
	return map[string]string{
		"type": "data",
		"name": "test",
	}
}

func TestReadLabels(t *testing.T) {

	// test happy path
	s := NewSidecar("test-data-1")
	s.pod.Labels = exampleLabels()

	err := s.readLabels()
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if expected, got := "test", s.couchbaseConfig.Name; expected != got {
		t.Errorf("unexpected couchbaseClusterName '%s', expected '%s'", got, expected)
	}

	if expected, got := []string{"kv"}, s.couchbaseConfig.Services; !reflect.DeepEqual(expected, got) {
		t.Errorf("unexpected couchbaseServices '%+v', expected '%+v'", got, expected)
	}

	//.test uppercase characters
	s.pod.Labels = map[string]string{
		"type": "DATA",
		"name": "teSt",
	}

	err = s.readLabels()
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if expected, got := "test", s.couchbaseConfig.Name; expected != got {
		t.Errorf("unexpected couchbaseClusterName '%s', expected '%s'", got, expected)
	}

	if expected, got := []string{"kv"}, s.couchbaseConfig.Services; !reflect.DeepEqual(expected, got) {
		t.Errorf("unexpected couchbaseServices '%+v', expected '%+v'", got, expected)
	}

	// multiple services
	s.pod.Labels = exampleLabels()
	s.pod.Labels["type"] = "query,data"

	err = s.readLabels()
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	{
		expected, got := []string{"kv", "n1ql"}, s.couchbaseConfig.Services
		sort.Strings(expected)
		sort.Strings(got)

		if !reflect.DeepEqual(expected, got) {
			t.Errorf("unexpected couchbaseServices '%+v', expected '%+v'", got, expected)
		}
	}

	// error: no valid service
	s.pod.Labels = exampleLabels()
	s.pod.Labels["type"] = "doingnothing"
	err = s.readLabels()
	if err == nil || !strings.Contains(err.Error(), "single valid service") {
		t.Errorf("expected an error for not having a single valid service")
	}

	// error: no type label
	s.pod.Labels = exampleLabels()
	delete(s.pod.Labels, "type")
	err = s.readLabels()
	if err == nil || !strings.Contains(err.Error(), "label 'type' is not") {
		t.Errorf("expected an error for not having a 'type' label")
	}

	// error: no name label
	s.pod.Labels = exampleLabels()
	delete(s.pod.Labels, "name")
	err = s.readLabels()
	if err == nil || !strings.Contains(err.Error(), "label 'name' is not") {
		t.Errorf("expected an error for not having a 'name' label, got: %s", err)
	}
}

func TestReadConfigMap(t *testing.T) {
	s := NewSidecar("test-data-1")
	err := s.readConfigMap()
	if err != nil {
		t.Errorf("unexpected error")
	}

	if expected, got := "user1", s.couchbaseConfig.Username; expected != got {
		t.Errorf("unexpected Username '%s', expected '%s'", got, expected)
	}

	if expected, got := "password1", s.couchbaseConfig.Password; expected != got {
		t.Errorf("unexpected Password '%s', expected '%s'", got, expected)
	}

	if expected, got := 1024, s.couchbaseConfig.IndexMemoryLimit; expected != got {
		t.Errorf("unexpected IndexMemoryLimit '%d', expected '%d'", got, expected)
	}

	if expected, got := 512, s.couchbaseConfig.DataMemoryLimit; expected != got {
		t.Errorf("unexpected DataMemoryLimit '%d', expected '%d'", got, expected)
	}

	if expected, got := 2048, s.couchbaseConfig.QueryMemoryLimit; expected != got {
		t.Errorf("unexpected QueryMemoryLimit '%d', expected '%d'", got, expected)
	}

}
