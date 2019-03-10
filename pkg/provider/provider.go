package provider

// ServiceProvider provides a list of service endpoints
type ServiceProvider interface {
	// GetServices returns all services and nodes that the provider is aware of
	GetServices() (services []Service, nodes map[string][]Service, err error)
}

// Service represents a group of endpoints that serve the same application
type Service struct {
	Name        string            `yaml:"name"`
	Annotations map[string]string `yaml:"annotations"`
	Endpoints   []Endpoint        `yaml:"endpoints"`
}

// Services is a list of services
type Services []Service

// Endpoint represents a address/port combination
type Endpoint struct {
	Address string `yaml:"address"`
	// FIXME:
	// Annotations on the endpoint level should be used to control e.g. loadBalancingWeight, loadAssignmentPolicies
	// use-case: blue-green deployment w/ gradual rollout
	//           there's some more conceptual work to do fo this use-case
	Annotations map[string]string `yaml:"annotations"`
	Port        uint32            `yaml:"port"`
}

// Annotaion allows a endpoint to specify additional routing configuration
type Annotaion string

const (
	// AnnotationEnableRetry enables retry functionality
	AnnotationEnableRetry = "enable-retry"
	// AnnotationNumRetries controls the retry behavior of a route
	AnnotationNumRetries = "num-retries"

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
