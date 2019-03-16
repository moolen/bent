package provider

import (
	"fmt"
	"time"

	"github.com/gogo/protobuf/types"
	google_protobuf "github.com/gogo/protobuf/types"
	"github.com/moolen/bent/envoy/api/v2"
	"github.com/moolen/bent/envoy/api/v2/core"
	"github.com/moolen/bent/envoy/api/v2/listener"
	"github.com/moolen/bent/envoy/api/v2/route"
	accesslog "github.com/moolen/bent/envoy/config/accesslog/v2"
	v21 "github.com/moolen/bent/envoy/config/filter/accesslog/v2"
	filterfault "github.com/moolen/bent/envoy/config/filter/fault/v2"
	authz "github.com/moolen/bent/envoy/config/filter/http/ext_authz/v2"
	fault "github.com/moolen/bent/envoy/config/filter/http/fault/v2"
	hc "github.com/moolen/bent/envoy/config/filter/http/health_check/v2"
	hcm "github.com/moolen/bent/envoy/config/filter/network/http_connection_manager/v2"
	_type "github.com/moolen/bent/envoy/type"
	"github.com/moolen/bent/pkg/util"
)

// Listener is a builder type for an envoy v2.Listener
type Listener struct {
	envoyListener *v2.Listener
	hcm           *hcm.HttpConnectionManager
}

// ListenerConfig defines the listener behavior
type ListenerConfig struct {
	// Name specifies the name of the listener
	Name string
	// TracingOperation specifies the traffic direction
	TracingOperation hcm.HttpConnectionManager_Tracing_OperationName
	// TargetRoute specifies the route which is associated with this listener
	TargetRoute string
	// Address specifies the IP address the listener listens on
	Address string
	// Port specifies the port the listener listens on
	Port uint32
}

// AuthzConfig defines the behavior of the Authz HTTP Filter
type AuthzConfig struct {
	Cluster string
}

// FaultConfig defines the behavior of the fault HTTP filter
type FaultConfig struct {
	Enabled       bool
	DelayDuration time.Duration
	DelayChance   uint32
	AbortCode     uint32
	AbortChance   uint32
}

// NewListener constructs a new Listener
func NewListener(cfg ListenerConfig) *Listener {
	lis := &Listener{
		envoyListener: &v2.Listener{
			Name: cfg.Name,
			Address: core.Address{
				Address: &core.Address_SocketAddress{
					SocketAddress: &core.SocketAddress{
						Protocol: core.TCP,
						Address:  cfg.Address,
						PortSpecifier: &core.SocketAddress_PortValue{
							PortValue: cfg.Port,
						},
					},
				},
			},
			FilterChains: []listener.FilterChain{{
				Filters: []listener.Filter{},
			}},
		},
		hcm: createConnectionManager(cfg),
	}
	return lis
}

func createConnectionManager(cfg ListenerConfig) *hcm.HttpConnectionManager {
	return &hcm.HttpConnectionManager{
		CodecType:  hcm.AUTO,
		StatPrefix: fmt.Sprintf("%s_http", cfg.TargetRoute),
		// allow absolute urls to enable egress via HTTP_PROXY
		HttpProtocolOptions: &core.Http1ProtocolOptions{
			AllowAbsoluteUrl: &types.BoolValue{Value: true},
		},
		AccessLog: []*v21.AccessLog{{
			Name: util.FileAccessLog,
			ConfigType: &v21.AccessLog_TypedConfig{
				TypedConfig: util.MessageToAny(&accesslog.FileAccessLog{
					Path: "/tmp/access.log",
					AccessLogFormat: &accesslog.FileAccessLog_JsonFormat{
						JsonFormat: jsonLog,
					},
				}),
			},
		}},
		Tracing: &hcm.HttpConnectionManager_Tracing{
			OperationName: cfg.TracingOperation,
		},
		RouteSpecifier: &hcm.HttpConnectionManager_Rds{
			Rds: &hcm.Rds{
				RouteConfigName: cfg.TargetRoute,
				ConfigSource: core.ConfigSource{
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
				},
			},
		},
		HttpFilters: []*hcm.HttpFilter{
			&hcm.HttpFilter{
				Name: util.Router,
			},
		},
	}
}

// InjectFault prepends the fault injection filter into the http filter chain
// order matters!
func (l Listener) InjectFault(cfg FaultConfig) {
	if !cfg.Enabled {
		return
	}

	httpFault := &fault.HTTPFault{}

	if cfg.AbortChance > 0 && cfg.AbortCode > 0 {
		httpFault.Abort = &fault.FaultAbort{
			ErrorType: &fault.FaultAbort_HttpStatus{
				HttpStatus: cfg.AbortCode,
			},
			Percentage: &_type.FractionalPercent{
				Numerator:   cfg.AbortChance,
				Denominator: _type.FractionalPercent_HUNDRED,
			},
		}
	}

	if cfg.DelayChance > 0 && cfg.DelayDuration > 0 {
		httpFault.Delay = &filterfault.FaultDelay{
			Type: filterfault.FaultDelay_FIXED,
			FaultDelaySecifier: &filterfault.FaultDelay_FixedDelay{
				FixedDelay: &cfg.DelayDuration,
			},
			Percentage: &_type.FractionalPercent{
				Numerator:   cfg.DelayChance,
				Denominator: _type.FractionalPercent_HUNDRED,
			},
		}
	}

	// prepend filter
	l.hcm.HttpFilters = append([]*hcm.HttpFilter{&hcm.HttpFilter{
		Name: util.Fault,
		ConfigType: &hcm.HttpFilter_TypedConfig{
			TypedConfig: util.MessageToAny(httpFault),
		},
	}}, l.hcm.HttpFilters...)
}

// InjectAuthz prepends the authz filter to the http filter chain
// order matters!
func (l Listener) InjectAuthz(cfg AuthzConfig) {
	timeout := time.Millisecond * 125
	l.hcm.HttpFilters = append([]*hcm.HttpFilter{&hcm.HttpFilter{
		Name: util.HTTPExternalAuthorization,
		ConfigType: &hcm.HttpFilter_TypedConfig{
			TypedConfig: util.MessageToAny(&authz.ExtAuthz{
				FailureModeAllow: false,
				Services: &authz.ExtAuthz_HttpService{
					HttpService: &authz.HttpService{
						ServerUri: &core.HttpUri{
							Uri:     "http://authfoo:3000/authenticate",
							Timeout: &timeout,
							HttpUpstreamType: &core.HttpUri_Cluster{
								Cluster: cfg.Cluster,
							},
						},
					},
				},
			}),
		},
	}}, l.hcm.HttpFilters...)
}

// InjectHealthCheckCache prependas a http health check cache into the http filter chain
// order matters!
func (l Listener) InjectHealthCheckCache(cluster Cluster) {
	cfg := cluster.Config()
	//cacheTime := time.Second * 30
	l.hcm.HttpFilters = append([]*hcm.HttpFilter{&hcm.HttpFilter{
		Name: util.HealthCheck,
		ConfigType: &hcm.HttpFilter_TypedConfig{
			TypedConfig: util.MessageToAny(&hc.HealthCheck{
				// TODO: implement pass-through-less caching:
				// https://www.envoyproxy.io/docs/envoy/latest/api-v2/config/filter/http/health_check/v2/health_check.proto
				PassThroughMode: &types.BoolValue{
					Value: true,
				},
				CacheTime: &cfg.HealthCheck.CacheDuration,
				Headers: []*route.HeaderMatcher{
					{
						Name: ":path",
						HeaderMatchSpecifier: &route.HeaderMatcher_ExactMatch{
							ExactMatch: cfg.HealthCheck.Path,
						},
					},
				},
			}),
		},
	}}, l.hcm.HttpFilters...)
}

// Resource builds and returns the envoy v2.listener
func (l Listener) Resource() *v2.Listener {
	// for now, reset filters when using this func multiple times
	l.envoyListener.FilterChains[0].Filters = []listener.Filter{}

	// convert hcm & append it to the filter chain
	l.envoyListener.FilterChains[0].Filters = append(
		l.envoyListener.FilterChains[0].Filters,
		listener.Filter{
			Name: util.HTTPConnectionManager,
			ConfigType: &listener.Filter_TypedConfig{
				TypedConfig: util.MessageToAny(l.hcm),
			},
		},
	)
	return l.envoyListener
}

var (
	jsonLog = &google_protobuf.Struct{
		Fields: map[string]*google_protobuf.Value{
			"start_time":                &google_protobuf.Value{Kind: &google_protobuf.Value_StringValue{StringValue: "%START_TIME%"}},
			"method":                    &google_protobuf.Value{Kind: &google_protobuf.Value_StringValue{StringValue: "%REQ(:METHOD)%"}},
			"path":                      &google_protobuf.Value{Kind: &google_protobuf.Value_StringValue{StringValue: "%REQ(X-ENVOY-ORIGINAL-PATH?:PATH)%"}},
			"protocol":                  &google_protobuf.Value{Kind: &google_protobuf.Value_StringValue{StringValue: "%PROTOCOL%"}},
			"response_code":             &google_protobuf.Value{Kind: &google_protobuf.Value_StringValue{StringValue: "%RESPONSE_CODE%"}},
			"response_flags":            &google_protobuf.Value{Kind: &google_protobuf.Value_StringValue{StringValue: "%RESPONSE_FLAGS%"}},
			"bytes_received":            &google_protobuf.Value{Kind: &google_protobuf.Value_StringValue{StringValue: "%BYTES_RECEIVED%"}},
			"bytes_sent":                &google_protobuf.Value{Kind: &google_protobuf.Value_StringValue{StringValue: "%BYTES_SENT%"}},
			"duration":                  &google_protobuf.Value{Kind: &google_protobuf.Value_StringValue{StringValue: "%DURATION%"}},
			"upstream_service_time":     &google_protobuf.Value{Kind: &google_protobuf.Value_StringValue{StringValue: "%RESP(X-ENVOY-UPSTREAM-SERVICE-TIME)%"}},
			"x_forwarded_for":           &google_protobuf.Value{Kind: &google_protobuf.Value_StringValue{StringValue: "%REQ(X-FORWARDED-FOR)%"}},
			"user_agent":                &google_protobuf.Value{Kind: &google_protobuf.Value_StringValue{StringValue: "%REQ(USER-AGENT)%"}},
			"request_id":                &google_protobuf.Value{Kind: &google_protobuf.Value_StringValue{StringValue: "%REQ(X-REQUEST-ID)%"}},
			"authority":                 &google_protobuf.Value{Kind: &google_protobuf.Value_StringValue{StringValue: "%REQ(:AUTHORITY)%"}},
			"upstream_host":             &google_protobuf.Value{Kind: &google_protobuf.Value_StringValue{StringValue: "%UPSTREAM_HOST%"}},
			"upstream_cluster":          &google_protobuf.Value{Kind: &google_protobuf.Value_StringValue{StringValue: "%UPSTREAM_CLUSTER%"}},
			"upstream_local_address":    &google_protobuf.Value{Kind: &google_protobuf.Value_StringValue{StringValue: "%UPSTREAM_LOCAL_ADDRESS%"}},
			"downstream_local_address":  &google_protobuf.Value{Kind: &google_protobuf.Value_StringValue{StringValue: "%DOWNSTREAM_LOCAL_ADDRESS%"}},
			"downstream_remote_address": &google_protobuf.Value{Kind: &google_protobuf.Value_StringValue{StringValue: "%DOWNSTREAM_REMOTE_ADDRESS%"}},
			"requested_server_name":     &google_protobuf.Value{Kind: &google_protobuf.Value_StringValue{StringValue: "%REQUESTED_SERVER_NAME%"}},
		},
	}
)
