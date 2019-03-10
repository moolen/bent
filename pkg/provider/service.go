package provider

func (s Service) hasAnnotation(check string) bool {
	for key := range s.Annotations {
		if key == check {
			return true
		}
	}
	return false
}

func (s Services) hasAnnotation(check string) bool {
	for _, service := range s {
		for key := range service.Annotations {
			if key == check {
				return true
			}
		}
	}
	return false
}

// GetAnnotation returns the first annotation value thath matches the key
func (s Services) GetAnnotation(key string) string {
	for _, svc := range s {
		for k, v := range svc.Annotations {
			if k == key {
				return v
			}
		}
	}
	return ""
}
