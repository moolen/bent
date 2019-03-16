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
	var cc EndpointConfig

	cc.Weight = 64
	if _, ok := e.Annotations[AnnotaionEndpointWeight]; ok {
		weight := uint32(parseIntWithFallback(e.Annotations[AnnotaionEndpointWeight], 64))
		if weight > 0 && weight <= 128 {
			cc.Weight = weight
		} else {
			log.Warnf("endpoint %s has invalid weight: %d", e.Address, weight)
		}
	}

	return cc
}
