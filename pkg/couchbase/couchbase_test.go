package couchbase

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	c, err := New("http://127.0.0.1:1234")
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}

	if expected, got := uint16(1234), c.Port(); expected != got {
		t.Errorf("Unexpected value: expected '%d', got '%d'", expected, got)
	}

	c, err = New("http://127.0.0.1")
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}

	if expected, got := uint16(80), c.Port(); expected != got {
		t.Errorf("Unexpected value: expected '%d', got '%d'", expected, got)
	}
}

func TestCouchbase_Connect_HappyPath_Auth(t *testing.T) {
	user := "user1"
	password := "password1"
	headerAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", user, password)))

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != headerAuth {
			w.WriteHeader(401)
			return
		}

		if r.URL.Path == "/pools/default" {
			fmt.Fprint(w, `{"nodes":[{"thisNode": true}]}`)
			return
		}
	}))
	defer ts.Close()

	c, err := New(ts.URL)
	c.Username = user
	c.Password = password
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}

	err = c.Connect()
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
}

func TestCouchbase_Connect_HappyPath_NoAuth(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/pools/default" {
			fmt.Fprint(w, `{"nodes":[{"thisNode": true}]}`)
			return
		}
	}))
	defer ts.Close()

	c, err := New(ts.URL)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}

	err = c.Connect()
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
}

func TestCouchbase_Connect_Wrong_Auth(t *testing.T) {
	user := "user1"
	password := "password1"
	headerAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", user, password)))

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != headerAuth {
			w.WriteHeader(401)
			return
		}

		if r.URL.Path == "/pools/default" {
			fmt.Fprint(w, `{"nodes":[{"thisNode": true}]}`)
			return
		}
	}))
	defer ts.Close()

	c, err := New(ts.URL)
	c.Username = user
	c.Password = "wrong"
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}

	err = c.Connect()
	if err == nil || !strings.Contains(err.Error(), "Check user/password") {
		t.Errorf("Expected error about a wrong password")
	}
}
