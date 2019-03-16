package provider

// ServiceProvider abstracts the provider from the mesh implementation
type ServiceProvider interface {
	// GetClusters provides a list of endpoints per node
	GetClusters() (map[string][]Cluster, error)
}

const (

	// ------
	// cluster level annotations
	// ------

	// AnnotationHealthCheckPath specifies the HTTP Path for health-checks
	AnnotationHealthCheckPath = "healthcheck.path"
	// AnnotationHealthInterval specifies the health check interval in milliseconds
	AnnotationHealthInterval = "healthcheck.interval"
	// AnnotationHealthCacheDuration specifies the health check cache duration in milliseconds
	AnnotationHealthCacheDuration = "healthcheck.cache"
	// AnnotationHealthTimeout specifies the timeout of a health-check in milliseconds
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
)
