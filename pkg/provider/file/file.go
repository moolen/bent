package file

import (
	"io/ioutil"

	log "github.com/sirupsen/logrus"

	"gopkg.in/yaml.v2"

	"github.com/moolen/bent/pkg/provider"
)

// Provider is a service provider that returns endpoints
type Provider struct {
	path string
}

type schema struct {
	Clusters      []provider.Cluster            `yaml:"clusters"`
	LocalClusters map[string][]provider.Cluster `yaml:"local_clusters"`
}

// NewProvider returns a new file provider
func NewProvider(path string) (*Provider, error) {

	return &Provider{
		path: path,
	}, nil
}

func readConfig(path string) (*schema, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg schema
	err = yaml.Unmarshal(content, &cfg)
	return &cfg, err
}

// GetClusters implements the provider.ServiceProvider interface
func (p Provider) GetClusters() (services []provider.Cluster, nodes map[string][]provider.Cluster, err error) {
	cfg, err := readConfig(p.path)
	if err != nil {
		return
	}
	log.Debugf("nodes: %#v", cfg.LocalClusters)
	log.Debugf("services: %#v", services)
	return cfg.Clusters, cfg.LocalClusters, err
}
