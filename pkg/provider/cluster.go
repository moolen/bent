package provider

import (
	"time"
)

const (
	defaultHealthCheckPath     = "/healthz"
	defaultHealthTimeout       = 3000  // in ms
	defaultHealthInterval      = 10000 // in ms
	defaultHealthCacheDuration = 30000 // in ms
)

// Cluster represents a group of endpoints
type Cluster struct {
	Name      string     `yaml:"name"`
	Endpoints []Endpoint `yaml:"endpoints"`
}

// ClusterConfig defines the cluster behavior
// this config is filled by annotations
type ClusterConfig struct {
	HealthCheck    ClusterHealthCheckConfig
	CircuitBreaker ClusterCircuitBreakerConfig
	// for now, the cluster specifies the fault configuration
	// of the INGRESS traffic
	FaultConfig FaultConfig
}

// ClusterHealthCheckConfig defines the health-checking behavior of a cluster
type ClusterHealthCheckConfig struct {
	Path                string
	Timeout             time.Duration
	Interval            time.Duration
	CacheDuration       time.Duration
	Port                uint32
	ExpectedStatusLower int64
	ExpectedStatusUpper int64
}

// ClusterCircuitBreakerConfig defines the circuit-breaker behavior of a cluster
type ClusterCircuitBreakerConfig struct {
	MaxConnections     uint32
	MaxPendingRequests uint32
	MaxRequests        uint32
	MaxRetries         uint32
}

// Config parses the annotations of the cluster and return a cluster config
func (c Cluster) Config() ClusterConfig {
	merged := mergeAnnotations(c)
	return parseClusterAnnotations(merged)
}

// this func looks up values in the annotations of the cluster
// it will pre-fill sane default values
func parseClusterAnnotations(ann map[string]string) ClusterConfig {
	lower, upper := parseInt64RangeWithFallback(
		ann[AnnotationHealthExpectedStatus], 200, 400)

	cc := ClusterConfig{
		FaultConfig: FaultConfig{
			Enabled:       getBool(ann, AnnotaionFaultInject, false),
			DelayChance:   getUInt32(ann, AnnotaionFaultDelayPercent, 0),
			DelayDuration: getDurationMilliseconds(ann, AnnotaionFaultDelayDuration, 30),
			AbortChance:   getUInt32(ann, AnnotaionFaultAbortPercent, 0),
			AbortCode:     getUInt32(ann, AnnotaionFaultAbortCode, 503),
		},
		CircuitBreaker: ClusterCircuitBreakerConfig{
			MaxConnections:     getUInt32(ann, AnnotaionCBMaxConn, 1000),
			MaxPendingRequests: getUInt32(ann, AnnotaionCBMaxPending, 1000),
			MaxRequests:        getUInt32(ann, AnnotaionCBMaxRequests, 1000),
			MaxRetries:         getUInt32(ann, AnnotaionCBMaxRetries, 3),
		},
		HealthCheck: ClusterHealthCheckConfig{
			Timeout:             getDurationMilliseconds(ann, AnnotationHealthTimeout, defaultHealthTimeout),
			Interval:            getDurationMilliseconds(ann, AnnotationHealthInterval, defaultHealthInterval),
			CacheDuration:       getDurationMilliseconds(ann, AnnotationHealthCacheDuration, defaultHealthCacheDuration),
			Path:                getString(ann, AnnotationHealthCheckPath, defaultHealthCheckPath),
			Port:                getUInt32(ann, AnnotationHealthPort, 0),
			ExpectedStatusLower: lower,
			ExpectedStatusUpper: upper,
		},
	}

	return cc
}

func getUInt32(ann map[string]string, key string, fallback uint32) uint32 {
	if _, ok := ann[key]; ok {
		num := parseIntWithFallback(ann[key], -1)
		if num > 0 {
			return uint32(num)
		}
	}
	return fallback
}

func getDurationMilliseconds(ann map[string]string, key string, fallback int) time.Duration {
	if _, ok := ann[key]; ok {
		num := parseIntWithFallback(ann[key], -1)
		if num > 0 {
			return time.Millisecond * time.Duration(num)
		}
	}
	return time.Millisecond * time.Duration(fallback)
}

func getString(ann map[string]string, key string, fallback string) string {
	if val, ok := ann[key]; ok {
		return val
	}
	return fallback
}

// key set = true
func getBool(ann map[string]string, key string, fallback bool) bool {
	if _, ok := ann[key]; ok {
		return true
	}
	return fallback
}
