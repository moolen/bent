package provider

import log "github.com/sirupsen/logrus"

// Endpoint represents a address/port combination
type Endpoint struct {
	Address     string            `yaml:"address"`
	Annotations map[string]string `yaml:"annotations"`
	Port        uint32            `yaml:"port"`
}

// EndpointConfig defines the behavior of a endpoint
type EndpointConfig struct {
	Weight uint32
}

// Config parses the endpoints annotations and returns the endpoint config
func (e Endpoint) Config() EndpointConfig {
	return e.parseEndpointAnnotations()
}

// this func looks up values in the annotations of the endpoint
// it will pre-fill sane default values (weight must be: 0 < weight <= 128)
func (e Endpoint) parseEndpointAnnotations() EndpointConfig {
	weight := getUInt32(e.Annotations, AnnotaionEndpointWeight, 64)
	if weight == 0 || weight > 128 {
		weight = 64
		log.Warnf("weight of endpoint %s has invalid weight", e.Address)
	}
	cc := EndpointConfig{
		Weight: weight,
	}

	return cc
}
