package provider

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/gogo/protobuf/proto"
	log "github.com/sirupsen/logrus"

	"github.com/moolen/bent/envoy/api/v2/route"
	"github.com/moolen/bent/pkg/cache"
)

const (
	defaultEgressTrafficPort  = 4000
	defaultIngressTrafficPort = 4100
	defaultHealthCheckPath    = "/healthz"
	defaultHealthTimeout      = 3000  // in ms
	defaultHealthInterval     = 10000 // in ms

	localClusterPrefix = "local"

	egressRoute  = "egress_route"
	ingressRoute = "ingress_route"
)

// Updater is the glue between the provider-specific implementation
// and the snapshot cache
// The updater requests the services, endpoints and nodes from the provider,
// transforms them into the envoy typespace and puts them into the cache
type Updater struct {
	cache    cache.SnapshotCache
	provider ServiceProvider
}

// NewUpdater returns a new Updater
func NewUpdater(config cache.SnapshotCache, provider ServiceProvider) *Updater {
	return &Updater{
		cache:    config,
		provider: provider,
	}
}

// transform transforms the clusters from the provider into a []Node
// the caller is responsible to persist the data
func transform(providerClusters map[string][]Cluster) ([]*Node, error) {
	var globalCluster []Cluster
	var globalVHosts []route.VirtualHost
	var nodes []*Node

	// prep global cluster data
	for _, clusters := range providerClusters {
		globalCluster = append(globalCluster, makeEgressClusters(clusters)...)
		for _, cluster := range clusters {
			globalVHosts = append(globalVHosts, getVirtualHost(cluster.Name, cluster.Name, cluster.Annotations))
		}
	}

	for node, clusters := range providerClusters {
		node := NewNode(node)
		// global
		node.AddCluster(globalCluster...)
		node.AddVirtualHosts(egressRoute, globalVHosts...)
		for _, cluster := range clusters {
			// internal
			localClusterName := fmt.Sprintf("%s_%s",
				localClusterPrefix, cluster.Name)
			node.AddCluster(Cluster{
				Name:        localClusterName,
				Annotations: cluster.Annotations,
				Endpoints:   cluster.Endpoints,
			})
			node.AddVirtualHosts(ingressRoute, getVirtualHost(cluster.Name, localClusterName, cluster.Annotations))
		}

		listeners, err := makeListeners(Clusters(clusters))
		if err != nil {
			log.Errorf("error creating listeners: %s", err)
			return nodes, err
		}
		node.AddListener(listeners...)
		nodes = append(nodes, node)
	}

	// handle ingress
	node := NewNode("ingress")
	node.AddCluster(globalCluster...)
	node.AddVirtualHosts(ingressRoute, globalVHosts...)
	ingress, err := makeIngress(nil)
	if err != nil {
		return nil, err
	}
	node.AddListener(ingress)
	nodes = append(nodes, node)
	return nodes, nil
}

// Run continuously polls the provider for changes and updates the cache accordingly
// every node has its own configuration
func (a Updater) Run() {
	for {
		var nodes []*Node
		var snap cache.Snapshot
		providerEndpoints, err := a.provider.GetClusters()
		if err != nil {
			log.Errorf("error fetching globalCluster: %s", err)
			goto Wait
		}
		nodes, err = transform(providerEndpoints)
		if err != nil {
			log.Errorf("error transforming data: %s", err)
		}
		for _, node := range nodes {
			snap = cache.NewSnapshot(
				computeVersion(node.Endpoints()),
				node.Endpoints(),
				node.Clusters(),
				node.Routes(),
				node.Listeners(),
			)
			a.cache.SetSnapshot(node.Name, snap)
		}

	Wait:
		<-time.After(time.Second * 10)
	}
}

// computeVersion takes a bunch of resources
// and computes a hash using their protobuf representation
func computeVersion(resources []cache.Resource) string {
	hash := md5.New()
	for _, ep := range resources {
		b, err := proto.Marshal(ep)
		if err != nil {
			continue
		}
		hash.Write(b)
	}
	return hex.EncodeToString(hash.Sum(nil))
}
