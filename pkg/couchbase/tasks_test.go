package couchbase

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"testing"
)

func TestCouchbase_Tasks(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/pools/default/tasks" {
			fmt.Fprint(w, `[
   {
      "progress" : 10.3286579207819,
      "subtype" : "rebalance",
      "type" : "rebalance",
      "status" : "running",
      "recommendedRefreshPeriod" : 0.25,
      "perNode" : {
         "ns_1@couchbase-data-2.couchbase-data.default.svc.cluster.local" : {
            "progress" : 10.850439882698
         },
         "ns_1@couchbase-data-1.couchbase-data.default.svc.cluster.local" : {
            "progress" : 9.39334637964775
         },
         "ns_1@couchbase-data-0.couchbase-data.default.svc.cluster.local" : {
            "progress" : 10.7421875
         }
      }
   }
]`)
			return
		}
	}))
	defer ts.Close()

	c, err := New(ts.URL)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}

	tasks, err := c.Tasks()
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}

	if expected, got := 1, len(tasks); expected != got {
		t.Errorf("Unexpected value: expected '%d', got '%d'", expected, got)
	}

	if expected, got := "rebalance", tasks[0].Type; expected != got {
		t.Errorf("Unexpected value: expected '%s', got '%s'", expected, got)
	}

	if expected, got := "rebalance", tasks[0].SubType; expected != got {
		t.Errorf("Unexpected value: expected '%s', got '%s'", expected, got)
	}

	if expected, got := "running", tasks[0].Status; expected != got {
		t.Errorf("Unexpected value: expected '%s', got '%s'", expected, got)
	}

	if expected, got := 10.3286579207819, tasks[0].Progress; expected != got {
		t.Errorf("Unexpected value: expected '%f', got '%f'", expected, got)
	}

	if expected, got := 0.25, tasks[0].RecommendedRefreshPeriod; expected != got {
		t.Errorf("Unexpected value: expected '%f', got '%f'", expected, got)
	}

	if expected, got := 3, len(tasks[0].PerNode); expected != got {
		t.Errorf("Unexpected value: expected '%d', got '%d'", expected, got)
	}
}

func TestCouchbase_RebalanceStatus_Running(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/pools/default/tasks" {
			fmt.Fprint(w, `[
   {
      "progress" : 10.3286579207819,
      "subtype" : "rebalance",
      "type" : "rebalance",
      "status" : "running",
      "recommendedRefreshPeriod" : 0.25,
      "perNode" : {
         "ns_1@couchbase-data-2.couchbase-data.default.svc.cluster.local" : {
            "progress" : 10.850439882698
         },
         "ns_1@couchbase-data-1.couchbase-data.default.svc.cluster.local" : {
            "progress" : 9.39334637964775
         },
         "ns_1@couchbase-data-0.couchbase-data.default.svc.cluster.local" : {
            "progress" : 10.7421875
         }
      }
   }
]`)
			return
		}
	}))
	defer ts.Close()

	c, err := New(ts.URL)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}

	status, err := c.RebalanceStatus()
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}

	if expected, got := true, status.Running; expected != got {
		t.Errorf("Unexpected value: expected '%+v', got '%+v'", expected, got)
	}

	nodes := []string{
		"ns_1@couchbase-data-2.couchbase-data.default.svc.cluster.local",
		"ns_1@couchbase-data-1.couchbase-data.default.svc.cluster.local",
		"ns_1@couchbase-data-0.couchbase-data.default.svc.cluster.local",
	}
	sort.Strings(nodes)
	sort.Strings(status.Nodes)

	if expected, got := nodes, status.Nodes; !reflect.DeepEqual(expected, got) {
		t.Errorf("Unexpected value: expected '%+v', got '%+v'", expected, got)
	}
}

func TestCouchbase_RebalanceStatus_NotRunning(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/pools/default/tasks" {
			fmt.Fprint(w, `[{"type":"rebalance","status":"notRunning","statusIsStale":false,"masterRequestTimedOut":false}]`)
			return
		}
	}))
	defer ts.Close()

	c, err := New(ts.URL)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}

	status, err := c.RebalanceStatus()
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}

	if expected, got := false, status.Running; expected != got {
		t.Errorf("Unexpected value: expected '%+v', got '%+v'", expected, got)
	}
}
