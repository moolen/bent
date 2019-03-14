package provider

// ServiceProvider abstracts the provider from the mesh implementation
type ServiceProvider interface {
	// GetClusters provides a list of endpoints per node
	GetClusters() (map[string][]Cluster, error)
}

// Cluster represents a group of endpoints
type Cluster struct {
	Name        string            `yaml:"name"`
	Annotations map[string]string `yaml:"annotations"`
	Endpoints   []Endpoint        `yaml:"endpoints"`
}

// Clusters is a list of services
type Clusters []Cluster

// Endpoint represents a address/port combination
type Endpoint struct {
	Address     string            `yaml:"address"`
	Annotations map[string]string `yaml:"annotations"`
	Port        uint32            `yaml:"port"`
}

// AnnotationProvider provides annotation values
type AnnotationProvider interface {
	hasAnnotation(string) bool
	getAnnotation(string) string
}

const (

	// ------
	// cluster level annotations
	// ------

	// AnnotationHealthCheckPath specifies the HTTP Path for health-checks
	AnnotationHealthCheckPath = "healthcheck.path"
	// AnnotationHealthInterval specifies the health check interval in nanoseconds
	AnnotationHealthInterval = "healthcheck.interval"
	// AnnotationHealthTimeout specifies the timeout of a health-check in nanoseconds
	AnnotationHealthTimeout = "healthcheck.timeout"
	// AnnotationHealthPort specifies the tcp port for the health-check
	AnnotationHealthPort = "healthcheck.port"
	// AnnotationHealthExpectedStatus specifies the accepted status codes
	AnnotationHealthExpectedStatus = "healthcheck.expected-status"

	// AnnotaionCBMaxConn sets the maximum number of connections that Envoy will make to the upstream
	AnnotaionCBMaxConn = "circuit-breaker.max-connections"
	// AnnotaionCBMaxPending sets the maximum number of pending requests that Envoy will
	// allow to the upstream cluster
	AnnotaionCBMaxPending = "circuit-breaker.max-pending"
	// AnnotaionCBMaxRequests sets the maximum number of parallel requests
	AnnotaionCBMaxRequests = "circuit-breaker.max-requests"
	// AnnotaionCBMaxRetries sets maximum number of parallel retries that Envoy
	// will allow to the upstream cluster
	AnnotaionCBMaxRetries = "circuit-breaker.max-retries"

	// ------
	// endpoint level annotations
	// ------

	// AnnotationEnableRetry enables retry functionality
	AnnotationEnableRetry = "enable-retry"
	// AnnotationNumRetries controls the retry behavior of a route
	AnnotationNumRetries = "num-retries"

	// AnnotaionEndpointWeight specifies the loadbalancer weight of the endpoint
	AnnotaionEndpointWeight = "endpoint.weight"

	// ------
	// listener level annotations
	// ------

	// AnnotaionFaultInject enables fault injection
	AnnotaionFaultInject = "fault.inject"
	// AnnotaionFaultDelayPercent int value, specifies the delay injection percentage
	AnnotaionFaultDelayPercent = "fault.delay.percent"
	// AnnotaionFaultDelayDuration in milliseconds
	AnnotaionFaultDelayDuration = "fault.delay.duration"
	// AnnotaionFaultAbortPercent int value, specifies the abort injection percentage
	AnnotaionFaultAbortPercent = "fault.abort.percent"
	// AnnotaionFaultAbortCode specify the response status code
	AnnotaionFaultAbortCode = "fault.abort.code"

	// TODO: add healthcheck filter
	// https://www.envoyproxy.io/docs/envoy/latest/configuration/http_filters/health_check_filter
)
