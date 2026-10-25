package response

// User 用户信息响应。
type User struct {
	ID   uint64 `json:"id"`
	Name string `json:"name"`
	Age  int64  `json:"age"`
}

type LockInfo struct {
	Acquired bool `json:"acquired"`
}
