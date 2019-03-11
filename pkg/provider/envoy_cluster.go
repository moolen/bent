package provider

import (
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/moolen/bent/envoy/api/v2"
	"github.com/moolen/bent/envoy/api/v2/cluster"
	"github.com/moolen/bent/envoy/api/v2/core"
	_type "github.com/moolen/bent/envoy/type"
)

func makeCluster(name string, annotations map[string]string) *v2.Cluster {
	expectedStatusLower, expectedStatusUpper :=
		parseInt64RangeWithFallback(annotations[AnnotationHealthExpectedStatus], 200, 400)
	healthTimeout := time.Millisecond * time.Duration(
		parseIntWithFallback(annotations[AnnotationHealthTimeout], defaultHealthTimeout))
	healthInterval := time.Millisecond * time.Duration(
		parseIntWithFallback(annotations[AnnotationHealthInterval], defaultHealthInterval))

	healthCheckPath := defaultHealthCheckPath
	if _, ok := annotations[AnnotationHealthCheckPath]; ok {
		healthCheckPath = annotations[AnnotationHealthCheckPath]
	}

	// if nil, the endpoint port is used
	var healthCheckPort *types.UInt32Value
	if _, ok := annotations[AnnotationHealthPort]; ok {
		checkPort := parseIntWithFallback(annotations[AnnotationHealthPort], -1)
		if checkPort > 0 {
			healthCheckPort = &types.UInt32Value{Value: uint32(checkPort)}
		}
	}

	cb := &cluster.CircuitBreakers_Thresholds{
		// FIXME: use annotations to specify this settings
		Priority:           core.RoutingPriority_DEFAULT,
		MaxConnections:     &types.UInt32Value{Value: 1000},
		MaxPendingRequests: &types.UInt32Value{Value: 1000},
		MaxRequests:        &types.UInt32Value{Value: 1000},
		MaxRetries:         &types.UInt32Value{Value: 3},
	}

	if _, ok := annotations[AnnotaionCBMaxConn]; ok {
		num := parseIntWithFallback(annotations[AnnotaionCBMaxConn], -1)
		if num > 0 {
			cb.MaxConnections = &types.UInt32Value{Value: uint32(num)}
		}
	}

	if _, ok := annotations[AnnotaionCBMaxPending]; ok {
		num := parseIntWithFallback(annotations[AnnotaionCBMaxPending], -1)
		if num > 0 {
			cb.MaxPendingRequests = &types.UInt32Value{Value: uint32(num)}
		}
	}

	if _, ok := annotations[AnnotaionCBMaxRequests]; ok {
		num := parseIntWithFallback(annotations[AnnotaionCBMaxRequests], -1)
		if num > 0 {
			cb.MaxRequests = &types.UInt32Value{Value: uint32(num)}
		}
	}

	if _, ok := annotations[AnnotaionCBMaxRetries]; ok {
		num := parseIntWithFallback(annotations[AnnotaionCBMaxRetries], -1)
		if num > 0 {
			cb.MaxRetries = &types.UInt32Value{Value: uint32(num)}
		}
	}

	cluster := &v2.Cluster{
		Name:            name,
		ConnectTimeout:  1 * time.Second,
		Type:            v2.Cluster_EDS,
		DnsLookupFamily: v2.Cluster_V4_ONLY,
		LbPolicy:        v2.Cluster_ROUND_ROBIN,
		CircuitBreakers: &cluster.CircuitBreakers{
			Thresholds: []*cluster.CircuitBreakers_Thresholds{cb},
		},
		HealthChecks: []*core.HealthCheck{
			{
				Timeout:            &healthTimeout,
				Interval:           &healthInterval,
				UnhealthyThreshold: &types.UInt32Value{Value: 3},
				HealthyThreshold:   &types.UInt32Value{Value: 3},
				AltPort:            healthCheckPort,
				HealthChecker: &core.HealthCheck_HttpHealthCheck_{
					HttpHealthCheck: &core.HealthCheck_HttpHealthCheck{
						Path: healthCheckPath,
						ExpectedStatuses: []*_type.Int64Range{
							{
								Start: expectedStatusLower,
								End:   expectedStatusUpper,
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
