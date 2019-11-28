package helm

// This is taken from helm functionality of merging helm values
func MergeMaps(a, b map[interface{}]interface{}) map[interface{}]interface{} {
	out := make(map[interface{}]interface{}, len(a))
	for aKey, aValue := range a {
		out[aKey] = aValue
	}
	for bKey, bValue := range b {
		if bValue, ok := bValue.(map[interface{}]interface{}); ok {
			if outValue, ok := out[bKey]; ok {
				if outValue, ok := outValue.(map[interface{}]interface{}); ok {
					out[bKey] = MergeMaps(outValue, bValue)
					continue
				}
			}
		}
		out[bKey] = bValue
	}
	return out
}
