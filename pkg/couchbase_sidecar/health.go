package couchbase_sidecar

import (
	"fmt"
	"net"
	"net/http"

	"github.com/Sirupsen/logrus"
)

type healthCheck struct {
	cs *CouchbaseSidecar
}

func (m *healthCheck) Log() *logrus.Entry {
	return m.cs.Log().WithField("component", "healthCheck")
}

func (m *healthCheck) mux() *http.ServeMux {

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "couchbase-sidecar - 404 not found")
	})

	mux.HandleFunc("/_status/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")

		err := m.cs.CouchbaseLocalHealthy()
		if m.cs.Master() || err == nil {
			w.WriteHeader(http.StatusOK)
			m.Log().Debugf("Health check: ok")
			fmt.Fprint(w, "ok")
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			m.Log().Warnf("Failed health check: %s", err)
			fmt.Fprint(w, fmt.Sprintf("not ready: %s", err))
		}
	})

	return mux
}

func (m *healthCheck) run() {

	port := fmt.Sprintf(":%d", 8080)

	// listen on port
	listener, err := net.Listen("tcp", port)
	if err != nil {
		m.Log().Fatalf("error starting http server on %s: %s", port, err)
	}

	mux := m.mux()

	m.Log().Infof("server listening on http://%s/", port)

	// handle stop signal
	go func() {
		<-m.cs.stopCh
		m.Log().Infof("stopping server listening on http://%s/", port)
		listener.Close()
	}()

	http.Serve(listener, mux)
}
