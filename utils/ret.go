package utils

func SetRetVal(key string, val interface{}) map[string]interface{} {
	res := make(map[string]interface{})
	res[key] = val
	return res
}
