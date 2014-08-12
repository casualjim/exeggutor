package caprica

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/reverb/exeggutor/boatwright"
)

// CapricaClient a client to connect to a caprica instance
type CapricaClient struct {
	config *boatwright.HttpConfig
	client *http.Client
	URL    *url.URL
}

// New creates a new caprica http client
func New(config *boatwright.HttpConfig) *CapricaClient {
	u, err := url.Parse(config.URL)
	if err != nil {
		fmt.Errorf("Failed to parse caprica url, because: %v", err)
		os.Exit(1)
	}

	u.User = url.UserPassword(config.User, config.Password)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	return &CapricaClient{
		config: config,
		URL:    u,
		client: client,
	}
}

func contains(coll []string, value string) bool {
	for _, item := range coll {
		if strings.EqualFold(item, value) {
			return true
		}
	}
	return false
}

// GetInstances gets the instances for the specified cluster names and serviceNames
func (c *CapricaClient) GetInstances(clusterNames, serviceNames string) ([]CapricaService, error) {
	p := c.URL.Path
	if !strings.HasSuffix(p, "/") {
		p = p + "/"
	}
	if clusterNames != "" {
		c.URL.Query().Add("clusters", clusterNames)
	}
	if serviceNames != "" {
		c.URL.Query().Add("serviceNames", serviceNames)
	}

	c.URL.Path = p + "api/instances"
	req, err := http.NewRequest("GET", c.URL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json;charset=utf-8")
	pass, _ := c.URL.User.Password()
	req.SetBasicAuth(c.URL.User.Username(), pass)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data := CapricaInstancesResponse{}
	var dec = json.NewDecoder(resp.Body)
	if err := dec.Decode(&data); err != nil {
		return nil, err
	}

	// Caprica is a bit greedy in filtering hosts, we want to be specific
	clusters := strings.Split(clusterNames, ",")
	services := strings.Split(serviceNames, ",")
	var result []CapricaService
	for _, svc := range data.Result {
		if contains(services, svc.Name) && contains(clusters, svc.Cluster) {
			result = append(result, svc)
		}
	}

	return result, nil
}
