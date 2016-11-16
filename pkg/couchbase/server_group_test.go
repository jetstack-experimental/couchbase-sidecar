package couchbase

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCouchbase_ServerGroups(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/pools/default/serverGroups" {
			fmt.Fprint(w, `{
  "uri" : "/pools/default/serverGroups?rev=55602081",
  "groups" : [
    {
     "addNodeURI": "/pools/default/serverGroups/0/addNode",
     "uri": "/pools/default/serverGroups/0",
     "name": "Group 1"
    },
    {
     "addNodeURI": "/pools/default/serverGroups/1/addNode",
     "uri": "/pools/default/serverGroups/1",
     "name": "eu-west-1z"
    }
  ]
}`)
			return
		}
	}))
	defer ts.Close()

	c, err := New(ts.URL)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}

	serverGroups, err := c.ServerGroups()
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}

	if expected, got := 2, len(serverGroups.Groups); expected != got {
		t.Errorf("Unexpected value: expected '%d', got '%d'", expected, got)
	}

	if expected, got := "Group 1", serverGroups.Groups[0].Name; expected != got {
		t.Errorf("Unexpected value: expected '%s', got '%s'", expected, got)
	}

	if expected, got := "eu-west-1z", serverGroups.Groups[1].Name; expected != got {
		t.Errorf("Unexpected value: expected '%s', got '%s'", expected, got)
	}

	if expected, got := "/pools/default/serverGroups?rev=55602081", serverGroups.URI; expected != got {
		t.Errorf("Unexpected value: expected '%s', got '%s'", expected, got)
	}
}
