package provider

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/gogo/protobuf/proto"
	log "github.com/sirupsen/logrus"

	"github.com/moolen/bent/envoy/api/v2/route"
	hcm "github.com/moolen/bent/envoy/config/filter/network/http_connection_manager/v2"
	"github.com/moolen/bent/pkg/cache"
)

const (
	defaultEgressTrafficPort  = 4000
	defaultIngressTrafficPort = 4100

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
			globalVHosts = append(globalVHosts, createEnvoyVHost(VHostConfig{
				Hostname: cluster.Name,
				Cluster:  cluster.Name,
			}))
		}
	}

	for node, clusters := range providerClusters {
		node := NewNode(node)

		// global
		node.AddCluster(globalCluster...)
		node.AddRoute(egressRoute, globalVHosts...)

		ingressListener := NewListener(ListenerConfig{
			Address:          "0.0.0.0",
			Port:             defaultIngressTrafficPort,
			Name:             "default-ingress",
			TargetRoute:      ingressRoute,
			TracingOperation: hcm.INGRESS,
		})
		egressListener := NewListener(ListenerConfig{
			Address:          "0.0.0.0",
			Port:             defaultEgressTrafficPort,
			Name:             "default-egress",
			TargetRoute:      egressRoute,
			TracingOperation: hcm.EGRESS,
		})

		// internal cluster & endpoints
		for _, cluster := range clusters {
			// local clusters have a prefix like this: local_beta.svc
			localClusterName := fmt.Sprintf("%s_%s", localClusterPrefix, cluster.Name)

			node.AddCluster(Cluster{
				Name:      localClusterName,
				Endpoints: cluster.Endpoints,
			})
			node.AddRoute(ingressRoute, createEnvoyVHost(VHostConfig{
				Hostname: cluster.Name,
				Cluster:  localClusterName,
			}))

			ingressListener.InjectHealthCheckCache(cluster)
			ingressListener.InjectFault(cluster.Config().FaultConfig)
		}

		node.AddListener(ingressListener.Resource(), egressListener.Resource())
		nodes = append(nodes, node)
	}

	// handle ingress
	node := NewNode("ingress")
	node.AddCluster(globalCluster...)
	node.AddRoute(ingressRoute, globalVHosts...)
	ingressListener := NewListener(ListenerConfig{
		Address:          "0.0.0.0",
		Port:             defaultIngressTrafficPort,
		TargetRoute:      ingressRoute,
		TracingOperation: hcm.INGRESS,
	})
	ingressListener.InjectAuthz(AuthzConfig{
		Cluster: "authz",
	})
	node.AddListener(ingressListener.Resource())
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
