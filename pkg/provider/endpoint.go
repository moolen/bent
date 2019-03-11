package provider

func (ep Endpoint) getAnnotation(check string) string {
	for key, val := range ep.Annotations {
		if key == check {
			return val
		}
	}
	return ""
}

func (ep Endpoint) hasAnnotation(check string) bool {
	for key := range ep.Annotations {
		if key == check {
			return true
		}
	}
	return false
}
