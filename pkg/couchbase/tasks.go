package couchbase

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

type Task struct {
	Progress                 float64                 `json:"progress,omitempty"`
	SubType                  string                  `json:"subtype,omitempty"`
	Type                     string                  `json:"type,omitempty"`
	Status                   string                  `json:"status,omitempty"`
	StatusIsStale            bool                    `json:"statusIsStale,omitempty"`
	RecommendedRefreshPeriod float64                 `json:"recommendedRefreshPeriod,omitempty"`
	PerNode                  map[string]NodeProgress `json:"perNode,omitempty"`
}

type NodeProgress struct {
	Progress float64 `json:"progress,omitempty"`
}

type RebalanceStatus struct {
	Progress                 float64
	Nodes                    []string
	Running                  bool
	RecommendedRefreshPeriod float64
}

const RebalanceStatusNotRunning string = "notRunning"
const RebalanceStatusRunning string = "running"
const RebalanceStatusStale string = "stale"

func (c *Couchbase) RebalanceStatus() (*RebalanceStatus, error) {
	tasks, err := c.Tasks()
	if err != nil {
		return nil, err
	}

	for _, task := range tasks {
		if task.Type == "rebalance" {

			if task.Status == RebalanceStatusNotRunning {
				c.Log().Debug("no rebalance is running")
				return &RebalanceStatus{
					Running: false,
				}, nil
			}

			if task.Status == RebalanceStatusRunning {
				nodes := []string{}
				for node, _ := range task.PerNode {
					nodes = append(nodes, node)
				}
				status := &RebalanceStatus{
					Running:  true,
					Progress: task.Progress,
					Nodes:    nodes,
				}
				c.Log().Debug("rebalance status: %+v", status)
				return status, nil
			}
		}
	}

	return nil, fmt.Errorf("No rebalance status found")

}

func (c *Couchbase) Tasks() ([]Task, error) {
	resp, err := c.Request("GET", "/pools/default/tasks", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("Error while connecting: %s", err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var tasks []Task
	err = json.Unmarshal(body, &tasks)
	if err != nil {
		return nil, fmt.Errorf("Unexpected error: %s", err)
	}

	return tasks, nil
}
