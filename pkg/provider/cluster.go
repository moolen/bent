package provider

func (s Cluster) getAnnotation(check string) string {
	for key, val := range s.Annotations {
		if key == check {
			return val
		}
	}
	return ""
}

func (s Cluster) hasAnnotation(check string) bool {
	for key := range s.Annotations {
		if key == check {
			return true
		}
	}
	return false
}

func (s Clusters) hasAnnotation(check string) bool {
	for _, service := range s {
		if service.hasAnnotation(check) {
			return true
		}
	}
	return false
}

// getAnnotation returns the first annotation value thath matches the key
func (s Clusters) getAnnotation(key string) string {
	for _, svc := range s {
		for k, v := range svc.Annotations {
			if k == key {
				return v
			}
		}
	}
	return ""
}
