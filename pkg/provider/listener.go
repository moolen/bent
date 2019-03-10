package provider

import (
	"fmt"
	"log"
	"time"

	"github.com/gogo/protobuf/types"
	google_protobuf "github.com/gogo/protobuf/types"
	"github.com/moolen/bent/envoy/api/v2"
	"github.com/moolen/bent/envoy/api/v2/core"
	"github.com/moolen/bent/envoy/api/v2/listener"
	accesslog "github.com/moolen/bent/envoy/config/accesslog/v2"
	v21 "github.com/moolen/bent/envoy/config/filter/accesslog/v2"
	filterfault "github.com/moolen/bent/envoy/config/filter/fault/v2"
	fault "github.com/moolen/bent/envoy/config/filter/http/fault/v2"
	hcm "github.com/moolen/bent/envoy/config/filter/network/http_connection_manager/v2"
	_type "github.com/moolen/bent/envoy/type"
	"github.com/moolen/bent/pkg/cache"
	"github.com/moolen/bent/pkg/util"
)

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

func makeListeners(node string, services Services) ([]cache.Resource, error) {
	egressManager, err := makeHTTPManager(egressRoute, hcm.EGRESS, services)
	if err != nil {
		return nil, err
	}
	ingressManager, err := makeHTTPManager(ingressRoute, hcm.INGRESS, services)
	if err != nil {
		return nil, err
	}

	return []cache.Resource{
		&v2.Listener{
			Name: "egress-listener",
			Address: core.Address{
				Address: &core.Address_SocketAddress{
					SocketAddress: &core.SocketAddress{
						Protocol: core.TCP,
						Address:  "0.0.0.0",
						PortSpecifier: &core.SocketAddress_PortValue{
							PortValue: defaultEgressTrafficPort,
						},
					},
				},
			},
			FilterChains: []listener.FilterChain{{
				Filters: []listener.Filter{
					{
						Name: util.HTTPConnectionManager,
						ConfigType: &listener.Filter_TypedConfig{
							TypedConfig: egressManager,
						},
					},
				},
			}},
		},
		&v2.Listener{
			Name: "ingress-listener",
			Address: core.Address{
				Address: &core.Address_SocketAddress{
					SocketAddress: &core.SocketAddress{
						Protocol: core.TCP,
						Address:  "0.0.0.0",
						PortSpecifier: &core.SocketAddress_PortValue{
							PortValue: defaultIngressTrafficPort,
						},
					},
				},
			},
			FilterChains: []listener.FilterChain{{
				Filters: []listener.Filter{
					{
						Name: util.HTTPConnectionManager,
						ConfigType: &listener.Filter_TypedConfig{
							TypedConfig: ingressManager,
						},
					}},
			}},
		},
	}, nil
}

func makeHTTPManager(route string, tracingOperation hcm.HttpConnectionManager_Tracing_OperationName, services Services) (*types.Any, error) {
	var httpFilters []*hcm.HttpFilter

	logConfig, err := util.MessageToStruct(&accesslog.FileAccessLog{
		Path: "/tmp/access.log",
		AccessLogFormat: &accesslog.FileAccessLog_JsonFormat{
			JsonFormat: jsonLog,
		},
	})
	if err != nil {
		return nil, err
	}

	// add fault injection
	if services.hasAnnotation(AnnotaionFaultInject) {
		faultDuration := time.Millisecond * time.Duration(
			parseIntWithFallback(services.getAnnotation(AnnotaionFaultDelayDuration), 100))
		delayPercent := parseIntWithFallback(services.getAnnotation(AnnotaionFaultDelayPercent), 1)
		log.Printf("fault injection delay: %dms/%dpercent", faultDuration, delayPercent)

		abortCode := parseIntWithFallback(services.getAnnotation(AnnotaionFaultAbortCode), 503)
		abortPercent := parseIntWithFallback(services.getAnnotation(AnnotaionFaultAbortPercent), 1)
		log.Printf("fault injection abort. code: %d / %dpercent", abortCode, abortPercent)

		faultInjection := util.MessageToAny(&fault.HTTPFault{
			Delay: &filterfault.FaultDelay{
				Type: filterfault.FaultDelay_FIXED,
				FaultDelaySecifier: &filterfault.FaultDelay_FixedDelay{
					FixedDelay: &faultDuration,
				},
				Percentage: &_type.FractionalPercent{
					Numerator:   uint32(delayPercent),
					Denominator: _type.FractionalPercent_HUNDRED,
				},
			},
			Abort: &fault.FaultAbort{
				ErrorType: &fault.FaultAbort_HttpStatus{
					HttpStatus: uint32(abortCode),
				},
				Percentage: &_type.FractionalPercent{
					Numerator:   uint32(abortPercent),
					Denominator: _type.FractionalPercent_HUNDRED,
				},
			},
		})
		httpFilters = append(httpFilters, &hcm.HttpFilter{
			Name: util.Fault,
			ConfigType: &hcm.HttpFilter_TypedConfig{
				TypedConfig: faultInjection,
			},
		})

	}

	// FIXME: we need to cache ingress health-checks
	// healthCheckCacheDuration := time.Second * 90
	// healthCheckConfig := util.MessageToAny(&hcv2.HealthCheck{
	// 	PassThroughMode: &types.BoolValue{Value: true},
	// 	CacheTime:       &healthCheckCacheDuration,
	// })

	httpFilters = append(httpFilters, &hcm.HttpFilter{
		Name: util.Router,
	})

	return util.MessageToAny(&hcm.HttpConnectionManager{
		CodecType:  hcm.AUTO,
		StatPrefix: fmt.Sprintf("%s_http", route),
		// allow absolute urls to enable egress via HTTP_PROXY
		HttpProtocolOptions: &core.Http1ProtocolOptions{
			AllowAbsoluteUrl: &types.BoolValue{Value: true},
		},
		AccessLog: []*v21.AccessLog{{
			Name: util.FileAccessLog,
			ConfigType: &v21.AccessLog_Config{
				Config: logConfig,
			},
		}},
		Tracing: &hcm.HttpConnectionManager_Tracing{
			OperationName: tracingOperation,
		},
		RouteSpecifier: &hcm.HttpConnectionManager_Rds{
			Rds: &hcm.Rds{
				RouteConfigName: route,
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
		HttpFilters: httpFilters,
	}), nil
}
