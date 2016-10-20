package couchbase

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/Sirupsen/logrus"
)

type Couchbase struct {
	URL      *url.URL
	Username string
	Password string
}

type Node struct {
	Uptime               string   `json:"uptime,omitempty"`
	CouchApiBase         string   `json:"couchApiBase,omitempty"`
	ClusterMembership    string   `json:"clusterMembership,omitempty"`
	Status               string   `json:"status,omitempty"`
	ThisNode             bool     `json:"thisNode,omitempty"`
	Hostname             string   `json:"hostname,omitempty"`
	ClusterCompatibility int      `json:"clusterCompatibility,omitempty"`
	Version              string   `json:"version,omitempty"`
	OS                   string   `json:"os,omitempty"`
	Services             []string `json:"services,omitempty"`
}

type Pool struct {
	Nodes []Node `json:"nodes,omitempty"`
}

var ErrorNodeUninitialized error = fmt.Errorf("Node uninitialized")

func New(rawURL string) (*Couchbase, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	return &Couchbase{
		URL: u,
	}, nil
}

func (c *Couchbase) Request(method, path string, body io.Reader, header *http.Header) (resp *http.Response, err error) {
	client := &http.Client{}

	url := *c.URL
	url.Path = path

	req, err := http.NewRequest(method, url.String(), body)
	c.Log().Debugf("method=%s url=%s", method, url.String())
	if err != nil {
		return nil, err
	}
	if header != nil {
		req.Header = *header
	}
	return client.Do(req)
}

func (c *Couchbase) PostForm(path string, data url.Values) (resp *http.Response, err error) {
	headers := make(http.Header)
	headers.Set("Content-Type", "application/x-www-form-urlencoded")
	return c.Request("POST", path, strings.NewReader(data.Encode()), &headers)
}

func (c *Couchbase) CheckStatusCode(resp *http.Response, validStatusCodes []int) error {
	validStatusCodesString := make([]string, len(validStatusCodes))

	for i, statusCode := range validStatusCodes {
		if statusCode == resp.StatusCode {
			return nil
		}
		validStatusCodesString[i] = fmt.Sprintf("%d", statusCode)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf(
			"expected statusCode '%s', got %d: %s",
			strings.Join(validStatusCodesString, ", "),
			resp.StatusCode,
			err,
		)
	}

	return fmt.Errorf(
		"expected statusCode '%s', got %d: %s",
		strings.Join(validStatusCodesString, ", "),
		resp.StatusCode,
		string(body),
	)
}

func (c *Couchbase) Connect() error {
	info, err := c.Info()
	if err != nil {
		return err
	}
	c.Log().Debug("node infos: %+v", info)
	return err
}

func (c *Couchbase) Nodes() (nodes []Node, err error) {
	// connect without auth
	c.Log().Debugf("getting node information")
	resp, err := c.Request("GET", "/pools/default", nil, nil)
	if err != nil {
		return nodes, fmt.Errorf("Error while connecting: %s", err)
	}

	// connect with auth
	if resp.StatusCode == 401 {
		c.URL.User = url.UserPassword(c.Username, c.Password)
		resp, err = c.Request("GET", "/pools/default", nil, nil)
		if err != nil {
			return nodes, fmt.Errorf("Error while connecting: %s", err)
		}
	}

	// uninitialized
	if resp.StatusCode == 404 {
		return nodes, ErrorNodeUninitialized
	}

	err = c.CheckStatusCode(resp, []int{200})
	if err != nil {
		return nodes, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nodes, err
	}

	// parse json
	pool := Pool{}
	err = json.Unmarshal(body, &pool)
	if err != nil {
		return nodes, err
	}

	return pool.Nodes, nil
}

func (c *Couchbase) getInfo(nodes []Node) (*Node, error) {
	for _, node := range nodes {
		if node.ThisNode {
			return &node, nil
		}
	}
	return nil, fmt.Errorf("No node info found")
}

func (c *Couchbase) Info() (*Node, error) {
	nodes, err := c.Nodes()
	if err != nil {
		return nil, err
	}

	return c.getInfo(nodes)

}

func (c *Couchbase) Port() int {
	// TODO implement me
	return 8091
}

func (c *Couchbase) UpdateServices(services []string) error {
	c.Log().Debugf("update services to '%+v'", services)
	data := url.Values{}
	data.Set("services", strings.Join(services, ","))
	resp, err := c.PostForm("/node/controller/setupServices", data)
	if err != nil {
		return err
	}
	return c.CheckStatusCode(resp, []int{200})
}

func (c *Couchbase) UpdateMemoryDataQuota(quota int) error {
	return c.UpdateMemoryQuota("memoryQuota", quota)
}

func (c *Couchbase) UpdateMemoryIndexQuota(quota int) error {
	return c.UpdateMemoryQuota("indexMemoryQuota", quota)
}

func (c *Couchbase) UpdateMemoryQuota(key string, quota int) error {
	data := url.Values{}
	data.Set(key, fmt.Sprintf("%d", quota))
	resp, err := c.PostForm("/pools/default", data)
	if err != nil {
		return err
	}
	return c.CheckStatusCode(resp, []int{200})
}

func (c *Couchbase) Log() *logrus.Entry {
	return logrus.WithField("component", "couchbase")
}

func (c *Couchbase) UpdateHostname(hostname string) error {
	c.Log().Debugf("update hostname to '%s'", hostname)
	data := url.Values{}
	data.Set("hostname", hostname)
	resp, err := c.PostForm("/node/controller/rename", data)
	if err != nil {
		return err
	}
	return c.CheckStatusCode(resp, []int{200})
}

func (c *Couchbase) Ping() error {
	resp, err := c.Request("GET", "/settings/web", nil, nil)
	if err != nil {
		return err
	}
	return c.CheckStatusCode(resp, []int{200})
}

func (c *Couchbase) SetupAuth() error {
	resp, err := c.Request("GET", "/settings/web", nil, nil)
	if err != nil {
		return fmt.Errorf("Error while checking login: %s", err)
	}

	if resp.StatusCode == 200 {
		data := url.Values{}
		data.Set("username", c.Username)
		data.Set("password", c.Password)
		data.Set("port", fmt.Sprintf("%d", c.Port()))
		resp, err := c.PostForm("/settings/web", data)
		if err != nil {
			return err
		}
		err = c.CheckStatusCode(resp, []int{200})
		if err != nil {
			return err
		}
	} else if resp.StatusCode != 401 {
		return fmt.Errorf("Expected couchbase to respon with either 401 or 200")
	}

	return nil
}

func (c *Couchbase) Initialize(hostname string, services []string) error {
	err := c.UpdateHostname(hostname)
	if err != nil {
		return err
	}

	err = c.SetupAuth()
	if err != nil {
		return err
	}

	return nil
}

func (c *Couchbase) AddNode(nodeName, username, password string, services []string) error {
	data := url.Values{}
	data.Set("services", strings.Join(services, ","))
	data.Set("hostname", nodeName)
	data.Set("user", username)
	data.Set("password", password)
	data.Set("services", strings.Join(services, ","))
	resp, err := c.PostForm("/controller/addNode", data)
	if err != nil {
		return err
	}
	return c.CheckStatusCode(resp, []int{200})
}

func (c *Couchbase) Healthy() error {
	nodes, err := c.Nodes()
	if err != nil {
		return err
	}

	if len(nodes) < 2 {
		return fmt.Errorf("Node hasn't joined the cluster yet")
	}

	info, err := c.getInfo(nodes)
	if err != nil {
		return err
	}

	if got, expected := info.Status, "healthy"; got != expected {
		return fmt.Errorf("status of node is '%s', expected '%s'", got, expected)
	}

	return nil
}
