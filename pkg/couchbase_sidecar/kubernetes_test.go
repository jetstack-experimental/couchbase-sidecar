package couchbase_sidecar

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	kubeAPI "k8s.io/client-go/1.5/pkg/api/v1"
)

func NewSidecar(podName string) *CouchbaseSidecar {
	return &CouchbaseSidecar{
		PodName:      podName,
		PodNamespace: "mynamespace",
		pod:          &kubeAPI.Pod{},
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

	if expected, got := "test", s.couchbaseClusterName; expected != got {
		t.Errorf("unexpected couchbaseClusterName '%s', expected '%s'", got, expected)
	}

	if expected, got := []string{"data"}, s.couchbaseServices; !reflect.DeepEqual(expected, got) {
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

	if expected, got := "test", s.couchbaseClusterName; expected != got {
		t.Errorf("unexpected couchbaseClusterName '%s', expected '%s'", got, expected)
	}

	if expected, got := []string{"data"}, s.couchbaseServices; !reflect.DeepEqual(expected, got) {
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
		expected, got := []string{"data", "query"}, s.couchbaseServices
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
