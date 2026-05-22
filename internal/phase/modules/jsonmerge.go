package modules

import "encoding/json"

// deepMerge merges src into dst. Nested maps are merged recursively; other types
// (including slices) are overwritten wholesale.
func deepMerge(dst, src map[string]any) {
	for k, sv := range src {
		if sm, ok := sv.(map[string]any); ok {
			if dm, ok := dst[k].(map[string]any); ok {
				deepMerge(dm, sm)
				continue
			}
		}
		dst[k] = sv
	}
}

// jsonStr serializes v to a JSON string for comparison purposes.
func jsonStr(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
