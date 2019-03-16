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
func parseClusterAnnotations(annotations map[string]string) ClusterConfig {
	var cc ClusterConfig

	cc.HealthCheck.ExpectedStatusLower, cc.HealthCheck.ExpectedStatusUpper =
		parseInt64RangeWithFallback(annotations[AnnotationHealthExpectedStatus], 200, 400)

	cc.HealthCheck.Timeout = time.Millisecond * time.Duration(
		parseIntWithFallback(annotations[AnnotationHealthTimeout], defaultHealthTimeout))

	cc.HealthCheck.Interval = time.Millisecond * time.Duration(
		parseIntWithFallback(annotations[AnnotationHealthInterval], defaultHealthInterval))

	cc.HealthCheck.CacheDuration = time.Millisecond * time.Duration(
		parseIntWithFallback(annotations[AnnotationHealthCacheDuration], defaultHealthCacheDuration))

	cc.HealthCheck.Path = defaultHealthCheckPath
	if _, ok := annotations[AnnotationHealthCheckPath]; ok {
		cc.HealthCheck.Path = annotations[AnnotationHealthCheckPath]
	}

	if _, ok := annotations[AnnotationHealthPort]; ok {
		checkPort := parseIntWithFallback(annotations[AnnotationHealthPort], -1)
		if checkPort > 0 {
			cc.HealthCheck.Port = uint32(checkPort)
		}
	}

	// defaults
	cc.CircuitBreaker.MaxConnections = 1000
	cc.CircuitBreaker.MaxPendingRequests = 1000
	cc.CircuitBreaker.MaxRequests = 1000
	cc.CircuitBreaker.MaxRetries = 3

	if _, ok := annotations[AnnotaionCBMaxConn]; ok {
		num := parseIntWithFallback(annotations[AnnotaionCBMaxConn], -1)
		if num > 0 {
			cc.CircuitBreaker.MaxConnections = uint32(num)
		}
	}
	if _, ok := annotations[AnnotaionCBMaxPending]; ok {
		num := parseIntWithFallback(annotations[AnnotaionCBMaxPending], -1)
		if num > 0 {
			cc.CircuitBreaker.MaxPendingRequests = uint32(num)
		}
	}
	if _, ok := annotations[AnnotaionCBMaxRequests]; ok {
		num := parseIntWithFallback(annotations[AnnotaionCBMaxRequests], -1)
		if num > 0 {
			cc.CircuitBreaker.MaxRequests = uint32(num)
		}
	}
	if _, ok := annotations[AnnotaionCBMaxRetries]; ok {
		num := parseIntWithFallback(annotations[AnnotaionCBMaxRetries], -1)
		if num > 0 {
			cc.CircuitBreaker.MaxRetries = uint32(num)

		}
	}

	// fault
	if _, ok := annotations[AnnotaionFaultInject]; ok {
		cc.FaultConfig.Enabled = true
		cc.FaultConfig.AbortCode = 503
		cc.FaultConfig.DelayDuration = time.Millisecond * 30
	}
	if _, ok := annotations[AnnotaionFaultDelayPercent]; ok {
		num := parseIntWithFallback(annotations[AnnotaionFaultDelayPercent], -1)
		if num > 0 {
			cc.FaultConfig.DelayChance = uint32(num)
		}
	}
	if _, ok := annotations[AnnotaionFaultDelayDuration]; ok {
		num := parseIntWithFallback(annotations[AnnotaionFaultDelayDuration], -1)
		if num > 0 {
			cc.FaultConfig.DelayDuration = time.Millisecond * time.Duration(num)
		}
	}
	if _, ok := annotations[AnnotaionFaultAbortPercent]; ok {
		num := parseIntWithFallback(annotations[AnnotaionFaultAbortPercent], -1)
		if num > 0 {
			cc.FaultConfig.AbortChance = uint32(num)
		}
	}
	if _, ok := annotations[AnnotaionFaultAbortCode]; ok {
		num := parseIntWithFallback(annotations[AnnotaionFaultAbortCode], -1)
		if num > 0 {
			cc.FaultConfig.AbortCode = uint32(num)
		}
	}

	return cc
}
