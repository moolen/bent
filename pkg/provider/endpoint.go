package provider

import (
	"fmt"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/moolen/bent/envoy/api/v2/core"
	"github.com/moolen/bent/envoy/api/v2/endpoint"
	"github.com/moolen/bent/envoy/api/v2/route"
	log "github.com/sirupsen/logrus"
)

func getVirtualHost(dns, cluster string, annotations map[string]string) route.VirtualHost {
	log.Infof("vhost annotations: %#v", annotations)
	numRetries := parseIntWithFallback(annotations[AnnotationNumRetries], 3)

	// should be the normal 99th percentile latency
	retryTimeout := 500 * time.Millisecond

	vhost := route.VirtualHost{
		Name: fmt.Sprintf("vhost_%s", dns),
		Domains: []string{
			dns,
		},
		Routes: []route.Route{
			{
				Match: route.RouteMatch{
					PathSpecifier: &route.RouteMatch_Prefix{
						Prefix: "/",
					},
					// FIXME: specify HTTP Method filter defined by annotations
					// Headers: []*route.HeaderMatcher{
					// 	{
					// 		Name: ":method",
					// 		HeaderMatchSpecifier: &route.HeaderMatcher_ExactMatch{
					// 			ExactMatch: "GET",
					// 		},
					// 	},
					// },
				},
				Action: &route.Route_Route{
					Route: &route.RouteAction{
						ClusterSpecifier: &route.RouteAction_Cluster{
							Cluster: cluster,
						},
					},
				},
			},
		},
	}

	if _, ok := annotations[AnnotationEnableRetry]; ok {
		vhost.RetryPolicy = &route.RetryPolicy{
			RetryOn:       "5xx",
			NumRetries:    &types.UInt32Value{Value: uint32(numRetries)},
			PerTryTimeout: &retryTimeout,
		}
	}

	return vhost
}

func createEnvoyEndpoint(endpoints []Endpoint) []endpoint.LbEndpoint {
	var envoyEndpoints []endpoint.LbEndpoint
	for _, ep := range endpoints {
		envoyEndpoints = append(envoyEndpoints, endpoint.LbEndpoint{
			// FIXME: implement LoadBalancingWeight
			//        using endpoint-level annotations
			Metadata: &core.Metadata{},
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
