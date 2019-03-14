package provider

import (
	"fmt"
	"strconv"
	"testing"
	"time"

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
		"alpha.1": []Cluster{
			{
				Name: "alpha.svc",
				Annotations: map[string]string{
					AnnotationHealthInterval: "60000",
				},
				Endpoints: []Endpoint{
					{
						Address: "1.1.1.1",
						Port:    1312,
					},
				},
			},
		},
		"alpha.2": []Cluster{
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
		"beta.1": []Cluster{
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
		"beta.2": []Cluster{
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
		"alpha.1": map[string][]Cluster{
			"local": []Cluster{
				{
					Name: "local_alpha.svc",
					Annotations: map[string]string{
						AnnotationHealthInterval: "60000",
					},
					Endpoints: []Endpoint{
						{
							Address: "1.1.1.1",
							Port:    1312,
						},
					},
				},
			},
			"global": []Cluster{
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
		"alpha.2": map[string][]Cluster{
			"local": []Cluster{
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
			"global": []Cluster{
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
		"beta.1": map[string][]Cluster{
			"local": []Cluster{
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
			"global": []Cluster{
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
		if err := checkHealthAnnotation(node, cluster.Name, cluster.Annotations); err != nil {
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

func checkHealthAnnotation(node *Node, cluster string, annotations map[string]string) error {
	check := node.clusters[cluster].HealthChecks[0]
	checker := check.HealthChecker.(*core.HealthCheck_HttpHealthCheck_).HttpHealthCheck
	path := annotations[AnnotationHealthCheckPath]

	timeoutNum, _ := strconv.Atoi(annotations[AnnotationHealthTimeout])
	timeout := time.Millisecond * time.Duration(timeoutNum)

	intervalNum, _ := strconv.Atoi(annotations[AnnotationHealthInterval])
	interval := time.Millisecond * time.Duration(intervalNum)
	if path != "" && checker.Path != path {
		return fmt.Errorf("cluster %s has wrong path: expected %s, got %s", cluster, path, checker.Path)
	}
	if timeout != 0 && check.Timeout.Nanoseconds() != timeout.Nanoseconds() {
		return fmt.Errorf("cluster %s has wrong timeout: expected %#v, got %#v", cluster, check.Timeout.Nanoseconds(), timeout.Nanoseconds())
	}
	if interval != 0 && check.Interval.Nanoseconds() != interval.Nanoseconds() {
		return fmt.Errorf("cluster %s has wrong interval: expected %#v, got %#v", cluster, check.Interval.Nanoseconds(), interval.Nanoseconds())
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

	for i, ep := range eps {
		if _, ok := node.endpoints[name]; !ok {
			return fmt.Errorf("missing endpoint with key %s, found: %#v name", name, node.endpoints)
		}
		if node.endpoints[name].ClusterName != name {
			return fmt.Errorf("wrong cluster name! expecting %s, found: %s ", name, node.endpoints[name].ClusterName)
		}
		if node.endpoints[name].ClusterName != name {
			return fmt.Errorf("wrong cluster name! expecting %s, found: %s ", name, node.endpoints[name].ClusterName)
		}

		if !hasAddress(node.endpoints[name].Endpoints[0].LbEndpoints[i], ep.Address) {
			return fmt.Errorf("endpoint has wrong address. expected %s found: %#v ", ep.Address, getAddress(node.endpoints[name].Endpoints[0].LbEndpoints[i]))
		}
		if !hasPort(node.endpoints[name].Endpoints[0].LbEndpoints[i], ep.Port) {
			return fmt.Errorf("endpoint has wrong port. expected %d, found: %d ", ep.Port, getPort(node.endpoints[name].Endpoints[0].LbEndpoints[i]))
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

func hasAddress(ep endpoint.LbEndpoint, addr string) bool {
	return addr == getAddress(ep)
}

func hasPort(ep endpoint.LbEndpoint, port uint32) bool {
	return port == getPort(ep)
}
