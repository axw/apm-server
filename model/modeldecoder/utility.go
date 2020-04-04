package modeldecoder

import "encoding/json"

func getObject(obj map[string]interface{}, key string) map[string]interface{} {
	value, _ := obj[key].(map[string]interface{})
	return value
}

func decodeString(obj map[string]interface{}, key string, out *string) bool {
	if value, ok := obj[key].(string); ok {
		*out = value
		return true
	}
	return false
}

func decodeInt(obj map[string]interface{}, key string, out *int) bool {
	switch value := obj[key].(type) {
	case json.Number:
		if f, err := value.Float64(); err == nil {
			*out = int(f)
		}
	case float64:
		*out = int(value)
		return true
	}
	return false
}
