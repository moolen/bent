package provider

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/gogo/protobuf/proto"
	log "github.com/sirupsen/logrus"

	"github.com/moolen/bent/envoy/api/v2"
	"github.com/moolen/bent/envoy/api/v2/endpoint"
	"github.com/moolen/bent/envoy/api/v2/route"
	"github.com/moolen/bent/pkg/cache"
)

const (
	defaultEgressTrafficPort  = 4000
	defaultIngressTrafficPort = 4100
	defaultHealthCheckPath    = "/healthz"
	defaultHealthTimeout      = 3000  // in ms
	defaultHealthInterval     = 10000 // in ms
	localClusterName          = "local_cluster"

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

// Run continuously polls the provider for changes and updates the cache accordingly
// every node has its own configuration
func (a Updater) Run() {
	for {
		var endpoints []cache.Resource
		var listeners []cache.Resource
		var clusters []cache.Resource
		var routes []cache.Resource
		var services []Service
		var snap cache.Snapshot
		var err error
		var egressVHosts []route.VirtualHost
		nodes := make(map[string][]Service)

		services, nodes, err = a.provider.GetServices()
		if err != nil {
			log.Errorf("error fetching services: %s", err)
			goto Wait
		}

		for _, svc := range services {

			// define egress endpoints
			endpoints = append(endpoints, &v2.ClusterLoadAssignment{
				ClusterName: svc.Name,
				Endpoints: []endpoint.LocalityLbEndpoints{
					endpoint.LocalityLbEndpoints{
						LbEndpoints: createEnvoyEndpoint(svc),
					},
				},
			})

			// define egress clusters
			clusters = append(clusters, makeCluster(svc.Name, svc.Annotations))
			egressVHosts = append(egressVHosts, getVirtualHost(svc.Name, svc.Name, svc.Annotations))
		}

		// define egress routes
		routes = append(routes,
			&v2.RouteConfiguration{
				Name:         egressRoute,
				VirtualHosts: egressVHosts,
			},
		)

		// each node needs a local configuration that points to the sidecar-ed application
		for node, localSvcs := range nodes {

			for _, localSvc := range localSvcs {
				log.Debugf("adding local config for %s/%s", node, localSvc.Name)

				// add a local cluster
				clusters = append(clusters, makeCluster(fmt.Sprintf("%s_%s", localClusterName, localSvc.Name), localSvc.Annotations))

				// add local endpoints
				endpoints = append(endpoints, &v2.ClusterLoadAssignment{
					ClusterName: fmt.Sprintf("%s_%s", localClusterName, localSvc.Name),
					Endpoints: []endpoint.LocalityLbEndpoints{
						endpoint.LocalityLbEndpoints{
							LbEndpoints: createEnvoyEndpoint(localSvc),
						},
					},
				})

				// add route that points to local cluster
				routes = append(routes, &v2.RouteConfiguration{
					Name: ingressRoute,
					VirtualHosts: []route.VirtualHost{
						getVirtualHost(localSvc.Name, fmt.Sprintf("%s_%s", localClusterName, localSvc.Name), localSvc.Annotations),
					},
				})
			}
			listeners, err = makeListeners(node, localSvcs)
			if err != nil {
				log.Errorf("error creating listeners: %s", err)
				goto Wait
			}
			snap = cache.NewSnapshot(computeVersion(endpoints), endpoints, clusters, routes, listeners)
			a.cache.SetSnapshot(node, snap)
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
