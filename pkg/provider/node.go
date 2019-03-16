package provider

import (
	"fmt"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/moolen/bent/envoy/api/v2"
	"github.com/moolen/bent/envoy/api/v2/cluster"
	"github.com/moolen/bent/envoy/api/v2/core"
	"github.com/moolen/bent/envoy/api/v2/endpoint"
	"github.com/moolen/bent/envoy/api/v2/route"
	_type "github.com/moolen/bent/envoy/type"
	"github.com/moolen/bent/pkg/cache"
	log "github.com/sirupsen/logrus"
)

// Node holds the actual envoy-related data
// and acts only as a proxy type
type Node struct {
	Name        string
	clusters    map[string]*v2.Cluster
	endpoints   map[string]*v2.ClusterLoadAssignment
	routes      map[string]*v2.RouteConfiguration
	vhostExists map[string]map[string]struct{}
	listeners   []*v2.Listener
}

// NewNode constructs a new node
func NewNode(name string) *Node {
	return &Node{
		Name:        name,
		clusters:    make(map[string]*v2.Cluster),
		endpoints:   make(map[string]*v2.ClusterLoadAssignment),
		routes:      make(map[string]*v2.RouteConfiguration),
		vhostExists: make(map[string]map[string]struct{}),
	}
}

// AddEndpoints initializes a ClusterLoadAssigment and appends a bunch of endpoints to it
func (n *Node) AddEndpoints(cluster string, eps ...Endpoint) {
	if n.endpoints[cluster] == nil {
		n.endpoints[cluster] = &v2.ClusterLoadAssignment{
			ClusterName: cluster,
			Endpoints: []endpoint.LocalityLbEndpoints{
				{
					LbEndpoints: []endpoint.LbEndpoint{},
				},
			},
		}
	}
	n.endpoints[cluster].Endpoints[0].LbEndpoints = append(
		n.endpoints[cluster].Endpoints[0].LbEndpoints,
		createEnvoyEndpoint(eps)...,
	)
}

// AddCluster initializes a cluster and calls AddEndpoints
func (n *Node) AddCluster(clusters ...Cluster) {
	for _, c := range clusters {
		if n.clusters[c.Name] == nil {
			n.clusters[c.Name] = createEnvoyCluster(c)
		}
		n.AddEndpoints(c.Name, c.Endpoints...)
	}
}

// AddListener adds a bunch of listeners
func (n *Node) AddListener(lis ...*v2.Listener) {
	n.listeners = append(n.listeners, lis...)
}

// AddRoute initializes a route config and appends a bunch of vhosts to it
// a vhost is unique per route and must not be duplicated
// this function takes care of it
func (n *Node) AddRoute(routeName string, vhosts ...route.VirtualHost) {
	if n.routes[routeName] == nil {
		n.routes[routeName] = &v2.RouteConfiguration{
			Name: routeName,
		}
	}
	for _, vhost := range vhosts {
		if n.vhostExists[routeName] == nil {
			n.vhostExists[routeName] = make(map[string]struct{})
		}
		// have we seen a vhost already? no? -> append
		if _, ok := n.vhostExists[routeName][vhost.Name]; !ok {
			n.vhostExists[routeName][vhost.Name] = struct{}{}
			n.routes[routeName].VirtualHosts = append(
				n.routes[routeName].VirtualHosts,
				vhost,
			)
		}
	}
}

func createEnvoyEndpoint(endpoints []Endpoint) []endpoint.LbEndpoint {
	var envoyEndpoints []endpoint.LbEndpoint

	for _, ep := range endpoints {
		cfg := ep.Config()
		log.Debugf("ep config: %#v", cfg)
		envoyEndpoints = append(envoyEndpoints, endpoint.LbEndpoint{
			LoadBalancingWeight: &types.UInt32Value{Value: cfg.Weight},
			Metadata:            &core.Metadata{},
			HostIdentifier: &endpoint.LbEndpoint_Endpoint{
				Endpoint: &endpoint.Endpoint{
					Address: &core.Address{
						Address: &core.Address_SocketAddress{
							SocketAddress: &core.SocketAddress{
								Protocol: core.TCP,
								Address:  ep.Address,
								PortSpecifier: &core.SocketAddress_PortValue{
									PortValue: ep.Port,
								},
							},
						},
					},
				},
			},
		})
	}
	return envoyEndpoints
}

func createEnvoyCluster(c Cluster) *v2.Cluster {
	clusterCfg := c.Config()
	log.Debugf("cluster config: %#v", clusterCfg)
	// if nil, the endpoint port is used
	var healthCheckPort *types.UInt32Value
	if clusterCfg.HealthCheck.Port != 0 {
		healthCheckPort = &types.UInt32Value{Value: clusterCfg.HealthCheck.Port}
	}

	cb := &cluster.CircuitBreakers_Thresholds{
		Priority:           core.RoutingPriority_DEFAULT,
		MaxConnections:     &types.UInt32Value{Value: clusterCfg.CircuitBreaker.MaxConnections},
		MaxPendingRequests: &types.UInt32Value{Value: clusterCfg.CircuitBreaker.MaxPendingRequests},
		MaxRequests:        &types.UInt32Value{Value: clusterCfg.CircuitBreaker.MaxRequests},
		MaxRetries:         &types.UInt32Value{Value: clusterCfg.CircuitBreaker.MaxRetries},
	}

	cluster := &v2.Cluster{
		Name:            c.Name,
		ConnectTimeout:  1 * time.Second,
		Type:            v2.Cluster_EDS,
		DnsLookupFamily: v2.Cluster_V4_ONLY,
		LbPolicy:        v2.Cluster_ROUND_ROBIN,
		CircuitBreakers: &cluster.CircuitBreakers{
			Thresholds: []*cluster.CircuitBreakers_Thresholds{cb},
		},
		// TODO: implement od
		// OutlierDetection: &cluster.OutlierDetection{},
		HealthChecks: []*core.HealthCheck{
			{
				Timeout:            &clusterCfg.HealthCheck.Timeout,
				Interval:           &clusterCfg.HealthCheck.Interval,
				UnhealthyThreshold: &types.UInt32Value{Value: 3},
				HealthyThreshold:   &types.UInt32Value{Value: 3},
				AltPort:            healthCheckPort,
				HealthChecker: &core.HealthCheck_HttpHealthCheck_{
					HttpHealthCheck: &core.HealthCheck_HttpHealthCheck{
						Path: clusterCfg.HealthCheck.Path,
						ExpectedStatuses: []*_type.Int64Range{
							{
								Start: clusterCfg.HealthCheck.ExpectedStatusLower,
								End:   clusterCfg.HealthCheck.ExpectedStatusUpper,
							},
						},
					},
				},
			},
		},
		EdsClusterConfig: &v2.Cluster_EdsClusterConfig{
			EdsConfig: createXDSConfigSource(),
		},
	}

	return cluster
}

func createXDSConfigSource() *core.ConfigSource {
	return &core.ConfigSource{
		ConfigSourceSpecifier: &core.ConfigSource_ApiConfigSource{
			ApiConfigSource: &core.ApiConfigSource{
				ApiType: core.ApiConfigSource_GRPC,
				GrpcServices: []*core.GrpcService{
					{
						TargetSpecifier: &core.GrpcService_EnvoyGrpc_{
							EnvoyGrpc: &core.GrpcService_EnvoyGrpc{
								ClusterName: "xds",
							},
						},
					},
				},
			},
		},
	}
}

// VHostConfig defines the VHost target cluster
type VHostConfig struct {
	Hostname string
	Cluster  string
}

func createEnvoyVHost(cfg VHostConfig) route.VirtualHost {
	vhost := route.VirtualHost{
		Name: fmt.Sprintf("vhost_%s", cfg.Hostname),
		Domains: []string{
			cfg.Hostname,
			fmt.Sprintf("%s:%d", cfg.Hostname, defaultIngressTrafficPort),
		},
		Routes: []route.Route{
			{
				Match: route.RouteMatch{
					PathSpecifier: &route.RouteMatch_Prefix{
						Prefix: "/",
					},
				},
				Action: &route.Route_Route{
					Route: &route.RouteAction{
						ClusterSpecifier: &route.RouteAction_Cluster{
							Cluster: cfg.Cluster,
						},
					},
				},
			},
		},
	}

	return vhost
}

// Endpoints returns the endpoints as cache.Resources
func (n *Node) Endpoints() (eps []cache.Resource) {
	for _, ep := range n.endpoints {
		eps = append(eps, ep)
	}
	return
}

// Clusters returns the clusters as cache.Resources
func (n *Node) Clusters() (cls []cache.Resource) {
	for _, c := range n.clusters {
		cls = append(cls, c)
	}
	return
}

// Routes returns the routes as cache.Resources
func (n *Node) Routes() (rs []cache.Resource) {
	for _, r := range n.routes {
		rs = append(rs, r)
	}
	return
}

// Listeners returns the listeners as cache.Resources
func (n *Node) Listeners() (ls []cache.Resource) {
	for _, l := range n.listeners {
		ls = append(ls, l)
	}
	return ls
}
