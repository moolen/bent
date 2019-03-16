package provider

import (
	"strconv"
	"strings"
)

func parseIntWithFallback(val string, fallback int) int {
	num, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return num
}

func parseInt64RangeWithFallback(val string, fallback1, fallback2 int64) (int64, int64) {
	list := strings.Split(val, "-")
	if len(list) != 2 {
		return fallback1, fallback2
	}
	num1, err := strconv.ParseInt(list[0], 10, 64)
	if err != nil {
		return fallback1, fallback2
	}
	num2, err := strconv.ParseInt(list[2], 10, 64)
	if err != nil {
		return fallback1, fallback2
	}
	return num1, num2
}

// MakeEgressEndpoints makes the endpoints point to the ingress port
func makeEgressEndpoints(in []Endpoint) (out []Endpoint) {
	for _, ep := range in {
		out = append(out, Endpoint{
			Address:     ep.Address,
			Annotations: ep.Annotations,
			Port:        defaultIngressTrafficPort,
		})
	}
	return out
}

// makeEgressClusters makes the endpoints point to the ingress port
func makeEgressClusters(in []Cluster) (out []Cluster) {
	for _, cluster := range in {
		out = append(out, Cluster{
			Name:      cluster.Name,
			Endpoints: makeEgressEndpoints(cluster.Endpoints),
		})
	}
	return out
}

func mergeAnnotations(cluster Cluster) map[string]string {
	out := make(map[string]string)
	for _, ep := range cluster.Endpoints {
		for k, v := range ep.Annotations {
			out[k] = v
		}
	}
	return out
}
