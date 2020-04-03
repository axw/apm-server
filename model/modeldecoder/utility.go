package modeldecoder

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
	if value, ok := obj[key].(int); ok {
		*out = value
		return true
	}
	return false
}
