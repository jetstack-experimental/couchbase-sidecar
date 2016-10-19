package couchbase

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	log "github.com/Sirupsen/logrus"
)

type Couchbase struct {
	URL      *url.URL
	Username string
	Password string
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
	log.Debugf("method=%s url=%s", method, url.String())
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
	// connect without auth
	resp, err := c.Request("GET", "/pools/default", nil, nil)
	if err != nil {
		return fmt.Errorf("Error while connecting: %s", err)
	}

	// connect with auth
	if resp.StatusCode == 401 {
		c.URL.User = url.UserPassword(c.Username, c.Password)
		resp, err = c.Request("GET", "/pools/default", nil, nil)
		if err != nil {
			return fmt.Errorf("Error while connecting: %s", err)
		}
	}

	// uninitialized
	if resp.StatusCode == 404 {
		return ErrorNodeUninitialized
	}
	return c.CheckStatusCode(resp, []int{200})
}

func (c *Couchbase) Port() int {
	// TODO implement me
	return 8091
}

func (c *Couchbase) UpdateServices(services []string) error {
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
