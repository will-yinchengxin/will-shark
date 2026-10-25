// Package mapx 提供 map 构造辅助。
package mapx

func SetRetVal(key string, val interface{}) map[string]interface{} {
	res := make(map[string]interface{})
	res[key] = val
	return res
}
