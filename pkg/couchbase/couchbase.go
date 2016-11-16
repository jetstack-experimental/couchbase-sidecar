package couchbase

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
)

type Couchbase struct {
	URL      *url.URL
	Username string
	Password string
	info     *Node
	cluster  *Cluster
}

type Node struct {
	Uptime               string   `json:"uptime,omitempty"`
	CouchApiBase         string   `json:"couchApiBase,omitempty"`
	ClusterMembership    string   `json:"clusterMembership,omitempty"`
	ClusterCompatibility int      `json:"clusterCompatibility,omitempty"`
	Status               string   `json:"status,omitempty"`
	ThisNode             bool     `json:"thisNode,omitempty"`
	Hostname             string   `json:"hostname,omitempty"`
	Version              string   `json:"version,omitempty"`
	OS                   string   `json:"os,omitempty"`
	Services             []string `json:"services,omitempty"`
	IndexMemoryQuota     int      `json:"indexMemoryQuota,omitempty"`
	MemoryQuota          int      `json:"memoryQuota,omitempty"`
	RebalanceStatus      string   `json:"rebalanceStatus,omitempty"`
	OTPCookie            string   `json:"otpCookie,omitempty"`
	OTPNode              string   `json:"otpNode,omitempty"`
}

var ErrorNodeUninitialized error = fmt.Errorf("Node uninitialized")

type Cluster struct {
	IsAdminCreds bool   `json:"isAdminCreds,omitempty"`
	IsEnterprise bool   `json:"isEnterprise,omitempty"`
	UUID         string `json:"uuid,omitempty"`
}

type Pool struct {
	Nodes []Node `json:"nodes,omitempty"`
}

func New(rawURL string) (*Couchbase, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	return &Couchbase{
		URL: u,
	}, nil
}

func (c *Couchbase) Request(method, path string, body []byte, header *http.Header) (resp *http.Response, err error) {

	resp, err = c.request(method, path, bytes.NewReader(body), header)
	if err != nil {
		return nil, fmt.Errorf("Error while connecting: %s", err)
	}

	// connect with auth
	if resp.StatusCode == 401 {
		c.URL.User = url.UserPassword(c.Username, c.Password)
		resp, err = c.request(method, path, bytes.NewReader(body), header)
		if err != nil {
			return nil, fmt.Errorf("Error while connecting with auth: %s", err)
		}
		if resp.StatusCode == 401 {
			return nil, fmt.Errorf("Error authenticating. Check user/password")
		}
	}

	return resp, nil
}

func strSliceContains(slice []string, item string) bool {
	for _, elem := range slice {
		if item == elem {
			return true
		}
	}
	return false
}

func (c *Couchbase) request(method, path string, body io.Reader, header *http.Header) (resp *http.Response, err error) {
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
	return c.Request("POST", path, []byte(data.Encode()), &headers)
}

func (c *Couchbase) RemoveNodes(removeNodes []string) error {
	ejectNodes, _, _, allNodes, err := c.GetOTPNodes(removeNodes, []string{}, []string{})
	if err != nil {
		return err
	}

	if len(ejectNodes) != len(removeNodes) {
		return fmt.Errorf("Some nodes specified to be removed are not part of the cluster")
	}

	err = c.Rebalance(allNodes, ejectNodes)
	if err != nil {
		return err
	}

	var sleep time.Duration = 0
	for {
		time.Sleep(sleep)

		status, err := c.RebalanceStatus()
		if err != nil {
			sleep = 500 * time.Millisecond
			c.Log().Warnf("Error while checking rebalance status: %s", err)
			continue
		}
		sleep = time.Duration(int64(status.RecommendedRefreshPeriod * float64(time.Second)))

		nodeInRebalance := false
		for _, node := range ejectNodes {
			if strSliceContains(status.Nodes, node) {
				nodeInRebalance = true
			}
		}

		if nodeInRebalance {
			continue
		}

		nodes, err := c.Nodes()
		if err != nil {
			c.Log().Warnf("Error while getting nodes: %s", err)
			continue
		}

		nodeInCluster := false
		for _, node := range nodes {
			if strSliceContains(ejectNodes, node.OTPNode) {
				nodeInCluster = true
			}
		}
		if nodeInCluster {
			continue
		}

		c.Log().Infof("rebalance finished")
	}

	return nil

}

func (c *Couchbase) GetOTPNodes(ejectNodes, failoverNode, reAddNode []string) (outEjectNodes, outFailoverNodes, outReAddNodes, outAllNodes []string, err error) {

	nodes, err := c.Nodes()
	if err != nil {
		return
	}

	for _, node := range nodes {
		if node.OTPNode == "" {
			err = fmt.Errorf("Unable to get OTP name for %+v", node)
			return
		}
		outAllNodes = append(outAllNodes, node.OTPNode)

		if strSliceContains(ejectNodes, node.Hostname) {
			outEjectNodes = append(outEjectNodes, node.OTPNode)
		}
	}

	return outEjectNodes, outFailoverNodes, outReAddNodes, outAllNodes, nil
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
	_, err := c.Info()
	return err
}

func (c *Couchbase) Nodes() (nodes []Node, err error) {
	// connect without auth
	c.Log().Debugf("getting node information")
	resp, err := c.Request("GET", "/pools/default", nil, nil)
	if err != nil {
		return nodes, fmt.Errorf("Error while connecting: %s", err)
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
	if c.info == nil {
		nodes, err := c.Nodes()
		if err != nil {
			return nil, err
		}
		info, err := c.getInfo(nodes)
		if err != nil {
			return nil, err
		}
		c.info = info
	}
	return c.info, nil
}

func (c *Couchbase) Port() uint16 {
	hostParts := strings.Split(c.URL.Host, ":")
	if len(hostParts) < 2 {
		return uint16(80)
	}

	port, err := strconv.ParseInt(hostParts[len(hostParts)-1], 10, 16)
	if err != nil {
		return uint16(80)
	}
	return uint16(port)
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

func (c *Couchbase) EnsureMemoryQuota(dataQuota int, indexQuota int) error {
	info, err := c.Info()
	if err != nil {
		return err
	}

	if info.MemoryQuota != dataQuota {
		err := c.updateMemoryQuota("memoryQuota", dataQuota)
		if err != nil {
			return err
		}
		c.info = nil
	}

	if info.IndexMemoryQuota != indexQuota {
		err := c.updateMemoryQuota("indexMemoryQuota", indexQuota)
		if err != nil {
			return err
		}
		c.info = nil
	}

	return nil
}

func (c *Couchbase) ClusterID() (string, error) {
	cluster, err := c.Cluster()
	if err != nil {
		return "", err
	}
	return cluster.UUID, nil
}

func (c *Couchbase) Rebalance(knownNodes, ejectedNodes []string) error {
	c.Log().Debugf("rebalance nodes ejected=%+v known=%+v", ejectedNodes, knownNodes)
	data := url.Values{}
	data.Set("ejectedNodes", strings.Join(ejectedNodes, ","))
	data.Set("knownNodes", strings.Join(knownNodes, ","))
	resp, err := c.PostForm("/controller/rebalance", data)
	if err != nil {
		return err
	}
	return c.CheckStatusCode(resp, []int{200})
}

func (c *Couchbase) Cluster() (*Cluster, error) {
	if c.cluster == nil {
		resp, err := c.Request("GET", "/pools", nil, nil)
		if err != nil {
			return nil, fmt.Errorf("Error while connecting: %s", err)
		}

		err = c.CheckStatusCode(resp, []int{200})
		if err != nil {
			return nil, err
		}

		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		// parse json
		cluster := Cluster{}
		err = json.Unmarshal(body, &cluster)
		if err != nil {
			return nil, err
		}
		c.cluster = &cluster
	}

	return c.cluster, nil

}

func (c *Couchbase) updateMemoryQuota(key string, quota int) error {
	c.Log().Debugf("update quota %s to %d", key, quota)
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
		return fmt.Errorf("Expected couchbase to respond with either 401 or 200")
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
	data.Set("hostname", nodeName)
	data.Set("user", username)
	data.Set("password", password)
	data.Set("services", strings.Join(services, ","))
	c.Log().Debugf(
		"adding node hostname='%s' username='%s' password='%s' services='%s'",
		nodeName,
		username,
		password,
		strings.Join(services, ","),
	)
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

	// TODO: This should involve a clusterID comparison
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
