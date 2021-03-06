package provider

import (
	"fmt"
	"testing"

	"github.com/moolen/bent/envoy/api/v2/core"
	"github.com/moolen/bent/envoy/api/v2/endpoint"
)

type TestProvider struct {
	Mock map[string][]Cluster
	Err  error
}

func (t TestProvider) GetClusters() (map[string][]Cluster, error) {
	return t.Mock, t.Err
}

func TestTransform(t *testing.T) {

	test := map[string][]Cluster{
		"alpha.1": {
			{
				Name: "alpha.svc",
				Endpoints: []Endpoint{
					{
						Address: "1.1.1.1",
						Port:    1312,
						Annotations: map[string]string{
							AnnotationHealthInterval: "60000",
						},
					},
				},
			},
		},
		"alpha.2": {
			{
				Name: "alpha.svc",
				Endpoints: []Endpoint{
					{
						Address: "1.1.1.2",
						Port:    1312,
					},
				},
			},
		},
		"beta.1": {
			{
				Name: "beta.svc",
				Endpoints: []Endpoint{
					{
						Address: "1.1.1.3",
						Port:    1312,
					},
				},
			},
		},
		"beta.2": {
			{
				Name: "beta.svc",
				Endpoints: []Endpoint{
					{
						Address: "1.1.1.4",
						Port:    1312,
					},
				},
			},
		},
	}

	expect := map[string]map[string][]Cluster{
		"alpha.1": {
			"local": {
				{
					Name: "local_alpha.svc",
					Endpoints: []Endpoint{
						{
							Address: "1.1.1.1",
							Port:    1312,
							Annotations: map[string]string{
								AnnotationHealthInterval: "60000",
							},
						},
					},
				},
			},
			"global": {
				{
					Name: "alpha.svc",
					Endpoints: []Endpoint{
						{
							Address: "1.1.1.1",
							Port:    defaultIngressTrafficPort,
						},
						{
							Address: "1.1.1.2",
							Port:    defaultIngressTrafficPort,
						},
					},
				},
				{
					Name: "beta.svc",
					Endpoints: []Endpoint{
						{
							Address: "1.1.1.3",
							Port:    defaultIngressTrafficPort,
						},
						{
							Address: "1.1.1.4",
							Port:    defaultIngressTrafficPort,
						},
					},
				},
			},
		},
		"alpha.2": {
			"local": {
				{
					Name: "local_alpha.svc",
					Endpoints: []Endpoint{
						{
							Address: "1.1.1.2",
							Port:    1312,
						},
					},
				},
			},
			"global": {
				{
					Name: "alpha.svc",
					Endpoints: []Endpoint{
						{
							Address: "1.1.1.1",
							Port:    defaultIngressTrafficPort,
						},
						{
							Address: "1.1.1.2",
							Port:    defaultIngressTrafficPort,
						},
					},
				},
				{
					Name: "beta.svc",
					Endpoints: []Endpoint{
						{
							Address: "1.1.1.3",
							Port:    defaultIngressTrafficPort,
						},
						{
							Address: "1.1.1.4",
							Port:    defaultIngressTrafficPort,
						},
					},
				},
			},
		},
		"beta.1": {
			"local": {
				{
					Name: "local_beta.svc",
					Endpoints: []Endpoint{
						{
							Address: "1.1.1.3",
							Port:    1312,
						},
					},
				},
			},
			"global": {
				{
					Name: "alpha.svc",
					Endpoints: []Endpoint{
						{
							Address: "1.1.1.1",
							Port:    defaultIngressTrafficPort,
						},
						{
							Address: "1.1.1.2",
							Port:    defaultIngressTrafficPort,
						},
					},
				},
				{
					Name: "beta.svc",
					Endpoints: []Endpoint{
						{
							Address: "1.1.1.3",
							Port:    defaultIngressTrafficPort,
						},
						{
							Address: "1.1.1.4",
							Port:    defaultIngressTrafficPort,
						},
					},
				},
			},
		},
	}

	nodes, err := transform(test)
	if err != nil {
		t.Error(err)
	}

	for _, node := range nodes {
		if err := checkNode(expect[node.Name], node); err != nil {
			t.Errorf("error checking node %s: %s\n%#v", node.Name, err, node)
		}
	}
}

func checkNode(test map[string][]Cluster, node *Node) error {

	for _, cluster := range test["local"] {
		if err := checkCluster(node, cluster.Name, cluster.Endpoints); err != nil {
			return err
		}
		if err := checkHealthAnnotation(node, cluster); err != nil {
			return err
		}
	}

	for _, cluster := range test["global"] {
		if err := checkCluster(node, cluster.Name, cluster.Endpoints); err != nil {
			return err
		}
	}
	return nil
}

func checkHealthAnnotation(node *Node, cluster Cluster) error {
	cfg := cluster.Config()
	check := node.clusters[cluster.Name].HealthChecks[0]
	checker := check.HealthChecker.(*core.HealthCheck_HttpHealthCheck_).HttpHealthCheck

	if cfg.HealthCheck.Path != "" && checker.Path != cfg.HealthCheck.Path {
		return fmt.Errorf("cluster %s has wrong path: expected %s, got %s", cluster.Name, cfg.HealthCheck.Path, checker.Path)
	}
	if cfg.HealthCheck.Timeout != 0 && check.Timeout.Nanoseconds() != cfg.HealthCheck.Timeout.Nanoseconds() {
		return fmt.Errorf("cluster %s has wrong timeout: expected %#v, got %#v", cluster.Name, check.Timeout.Nanoseconds(), cfg.HealthCheck.Timeout.Nanoseconds())
	}
	if cfg.HealthCheck.Interval != 0 && check.Interval.Nanoseconds() != cfg.HealthCheck.Interval.Nanoseconds() {
		return fmt.Errorf("cluster %s has wrong interval: expected %#v, got %#v", cluster.Name, check.Interval.Nanoseconds(), cfg.HealthCheck.Interval.Nanoseconds())
	}

	return nil
}

func checkCluster(node *Node, name string, eps []Endpoint) error {
	if _, ok := node.clusters[name]; !ok {
		return fmt.Errorf("missing cluster %s, found: %#v ", name, node.clusters)
	}
	if node.clusters[name].Name != name {
		return fmt.Errorf("wrong cluster name! expecting %s, found: %s ", name, node.clusters[name].Name)
	}

	if len(node.endpoints[name].Endpoints[0].LbEndpoints) != len(eps) {
		return fmt.Errorf("wrong number of endpoints for cluster %s. expecting %d, found: %d ", name, len(node.endpoints[name].Endpoints[0].LbEndpoints), len(eps))
	}

	if _, ok := node.endpoints[name]; !ok {
		return fmt.Errorf("missing endpoint with key %s, found: %#v name", name, node.endpoints)
	}
	if node.endpoints[name].ClusterName != name {
		return fmt.Errorf("wrong cluster name! expecting %s, found: %s ", name, node.endpoints[name].ClusterName)
	}
	if node.endpoints[name].ClusterName != name {
		return fmt.Errorf("wrong cluster name! expecting %s, found: %s ", name, node.endpoints[name].ClusterName)
	}

	for _, ep := range eps {
		found := false
		for _, lbEndpoint := range node.endpoints[name].Endpoints[0].LbEndpoints {

			epAddr := getAddress(lbEndpoint)
			epPort := getPort(lbEndpoint)

			if epAddr == ep.Address && epPort == ep.Port {
				found = true
			}
		}
		if found == false {
			return fmt.Errorf("Endpoint was not found: %#v", ep)
		}
	}
	return nil
}

func getAddress(ep endpoint.LbEndpoint) string {
	return ep.HostIdentifier.(*endpoint.LbEndpoint_Endpoint).Endpoint.Address.Address.(*core.Address_SocketAddress).SocketAddress.Address
}

func getPort(ep endpoint.LbEndpoint) uint32 {
	return ep.HostIdentifier.(*endpoint.LbEndpoint_Endpoint).Endpoint.Address.Address.(*core.Address_SocketAddress).SocketAddress.PortSpecifier.(*core.SocketAddress_PortValue).PortValue
}
