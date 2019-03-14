package provider

import (
	"github.com/moolen/bent/envoy/api/v2"
	"github.com/moolen/bent/envoy/api/v2/endpoint"
	"github.com/moolen/bent/envoy/api/v2/route"
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
			n.clusters[c.Name] = makeCluster(c.Name, c.Annotations)
		}
		n.AddEndpoints(c.Name, c.Endpoints...)
	}
}

// AddListener adds a bunch of listeners
func (n *Node) AddListener(lis ...*v2.Listener) {
	n.listeners = append(n.listeners, lis...)
}

// AddVirtualHosts initializes a route config and appends a bunch of vhosts to it
// a vhost is unique per route and must not be duplicated
// this function takes care of it
func (n *Node) AddVirtualHosts(routeName string, vhosts ...route.VirtualHost) {
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
	log.Debugf("out listeners: %d", len(n.listeners))
	for _, l := range n.listeners {
		ls = append(ls, l)
	}
	return ls
}
