package couchbase

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
)

type ServerGroups struct {
	URI    string        `json:"uri,omitempty"`
	Groups []ServerGroup `json:"groups,omitempty"`
}

type ServerGroup struct {
	URI        string `json:"uri,omitempty"`
	Name       string `json:"name,omitempty"`
	AddNodeURI string `json:"addNodeURI,omitempty"`
}

func (c *Couchbase) ServerGroups() (*ServerGroups, error) {
	resp, err := c.Request("GET", "/pools/default/serverGroups", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("Error while connecting: %s", err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var serverGroups ServerGroups
	err = json.Unmarshal(body, &serverGroups)
	if err != nil {
		return nil, fmt.Errorf("Unexpected error: %s", err)
	}

	return &serverGroups, nil
}

func (c *Couchbase) ServerGroupURI(serverGroupName string) (URI string, err error) {
	serverGroups, err := c.ServerGroups()
	if err != nil {
		return "", err
	}

	for _, serverGroup := range serverGroups.Groups {
		if serverGroup.Name == serverGroupName {
			return serverGroup.AddNodeURI, nil
		}
	}

	err = c.CreateServerGroup(serverGroupName)
	if err != nil {
		return "", err
	}

	return c.ServerGroupURI(serverGroupName)
}

func (c *Couchbase) CreateServerGroup(serverGroupName string) error {
	c.Log().Infof("create serverGroup '%s'", serverGroupName)
	data := url.Values{}
	data.Set("name", serverGroupName)
	resp, err := c.PostForm("/pools/default/serverGroups", data)
	if err != nil {
		return err
	}
	return c.CheckStatusCode(resp, []int{200})
}
