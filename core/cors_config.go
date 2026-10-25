package core

type corsConfig struct {
	AllowOrigins []string `mapstructure:"allowOrigins"`
}

// CorsAllowOrigins 读取 cors.allowOrigins，未配置时返回空（不放行任何跨域来源）
func CorsAllowOrigins() []string {
	raw, err := GetSingleConfig(CoreConfig, "cors", corsConfig{})
	if err != nil {
		return nil
	}
	if c, ok := raw.(*corsConfig); ok {
		return c.AllowOrigins
	}
	return nil
}
