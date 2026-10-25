// Package config 强类型配置：一次加载、启动时校验，支持环境变量覆盖。
//
// 加载顺序（后者覆盖前者）：代码默认值 → envconfig/<env>_config.yaml → 环境变量
// 环境变量命名：WS_ + 配置路径大写，"." 替换为 "_"，例如
//
//	mysql.willshark.password  →  WS_MYSQL_WILLSHARK_PASSWORD
//	http.port                 →  WS_HTTP_PORT
//
// 注意：环境变量只能覆盖配置文件中已存在（或有默认值）的键；viper 会把 map 的键转为小写。
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"willshark/consts"

	"github.com/spf13/viper"
)

const (
	EnvPrefix    = "WS"
	EnvAppEnv    = "APP_ENV"       // 运行环境，优先级高于命令行参数
	EnvConfigDir = "WS_CONFIG_DIR" // 配置目录，默认 envconfig
)

type Config struct {
	App      App              `mapstructure:"app"`
	HTTP     HTTP             `mapstructure:"http"`
	Log      Log              `mapstructure:"log"`
	MySQL    map[string]MySQL `mapstructure:"mysql"`
	Redis    map[string]Redis `mapstructure:"redis"`
	RocketMQ RocketMQ         `mapstructure:"rocketmq"`
	Tracing  Tracing          `mapstructure:"tracing"`
	CORS     CORS             `mapstructure:"cors"`
}

type App struct {
	Name    string `mapstructure:"name"`
	ID      string `mapstructure:"id"`
	Swagger bool   `mapstructure:"swagger"` // 是否暴露 /swagger，生产环境应关闭
	Env     string `mapstructure:"-"`       // 由加载参数决定，不从文件读取
}

type HTTP struct {
	Port              int           `mapstructure:"port"`
	ReadHeaderTimeout time.Duration `mapstructure:"readHeaderTimeout"`
	ReadTimeout       time.Duration `mapstructure:"readTimeout"`
	// WriteTimeout 对流式接口（SSE/大模型输出）需在 handler 内用 http.ResponseController 单独放宽
	WriteTimeout    time.Duration `mapstructure:"writeTimeout"`
	IdleTimeout     time.Duration `mapstructure:"idleTimeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdownTimeout"` // 优雅退出等待在途请求的上限
	// DrainWait 收到退出信号后先将 /readyz 置为 503 并等待该时长再关闭，
	// 让 K8s 摘除流量；K8s 环境建议 5s，本地开发为 0
	DrainWait      time.Duration `mapstructure:"drainWait"`
	MaxBodyBytes   int64         `mapstructure:"maxBodyBytes"`
	TrustedProxies []string      `mapstructure:"trustedProxies"`
	// AlwaysOK 为 true 时业务错误也返回 HTTP 200（仅通过 body 中的 code 区分），用于兼容旧前端
	AlwaysOK bool `mapstructure:"alwaysOK"`
	// LogRequestBody 访问日志中记录请求体（JSON，≤2KB，敏感字段脱敏）
	LogRequestBody bool `mapstructure:"logRequestBody"`
}

type Log struct {
	Level  string `mapstructure:"level"`  // debug | info | warn | error
	Format string `mapstructure:"format"` // json | text
}

type MySQL struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	Database        string        `mapstructure:"database"`
	MaxIdleConns    int           `mapstructure:"maxIdleConns"`
	MaxOpenConns    int           `mapstructure:"maxOpenConns"`
	ConnMaxLifetime time.Duration `mapstructure:"connMaxLifetime"`
	ConnMaxIdleTime time.Duration `mapstructure:"connMaxIdleTime"`
	SlowThreshold   time.Duration `mapstructure:"slowThreshold"` // 慢 SQL 阈值，超过则打 warn 日志
}

type Redis struct {
	Addr            string        `mapstructure:"addr"`
	Password        string        `mapstructure:"password"`
	DB              int           `mapstructure:"db"`
	PoolSize        int           `mapstructure:"poolSize"` // 0 使用 go-redis 默认值（10 * GOMAXPROCS）
	MinIdleConns    int           `mapstructure:"minIdleConns"`
	MaxIdleConns    int           `mapstructure:"maxIdleConns"`
	ConnMaxIdleTime time.Duration `mapstructure:"connMaxIdleTime"`
	DialTimeout     time.Duration `mapstructure:"dialTimeout"`
	ReadTimeout     time.Duration `mapstructure:"readTimeout"`
	WriteTimeout    time.Duration `mapstructure:"writeTimeout"`
}

type RocketMQ struct {
	Enable      bool              `mapstructure:"enable"`
	NameServers []string          `mapstructure:"nameServers"`
	GroupName   string            `mapstructure:"groupName"`
	Retry       int               `mapstructure:"retry"`
	Topics      map[string]string `mapstructure:"topics"` // 逻辑 tag → 实际 topic
}

// RealTopic 返回逻辑 tag 对应的实际 topic，未配置时返回 tag 本身
func (r RocketMQ) RealTopic(tag string) string {
	if t, ok := r.Topics[tag]; ok {
		return t
	}
	return tag
}

type Tracing struct {
	Enable      bool    `mapstructure:"enable"`
	Endpoint    string  `mapstructure:"endpoint"` // OTLP/HTTP host:port，如 127.0.0.1:4318
	Insecure    bool    `mapstructure:"insecure"`
	SampleRatio float64 `mapstructure:"sampleRatio"` // 0~1，根 span 采样率；上游已采样的请求始终继续采样
}

type CORS struct {
	AllowOrigins []string `mapstructure:"allowOrigins"` // "*" 表示任意来源但不允许携带 cookie
}

// ResolveEnv 运行环境：APP_ENV > 第一个命令行参数 > dev
func ResolveEnv(args []string) string {
	if env := os.Getenv(EnvAppEnv); env != "" {
		return env
	}
	if len(args) > 0 && args[0] != "" {
		return args[0]
	}
	return "dev"
}

// Load 加载并校验 <dir>/<env>_config.yaml，dir 默认 envconfig，可用 WS_CONFIG_DIR 覆盖
func Load(env string) (*Config, error) {
	dir := os.Getenv(EnvConfigDir)
	if dir == "" {
		dir = consts.ConfigPath
	}
	return LoadFile(env, filepath.Join(dir, env+consts.ConfigName+"."+consts.ConfigType))
}

func LoadFile(env, path string) (*Config, error) {
	v := viper.New()
	setDefaults(v)
	v.SetConfigFile(path)
	v.SetEnvPrefix(EnvPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	var c Config
	if err := v.Unmarshal(&c); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	c.App.Env = env
	c.fillDefaults()
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config %s: %w", path, err)
	}
	return &c, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("app.name", consts.APP_NAME)
	v.SetDefault("app.id", consts.APP_ID)
	v.SetDefault("http.port", 8899)
	v.SetDefault("http.readHeaderTimeout", "5s")
	v.SetDefault("http.readTimeout", "30s")
	v.SetDefault("http.writeTimeout", "30s")
	v.SetDefault("http.idleTimeout", "60s")
	v.SetDefault("http.shutdownTimeout", "25s")
	v.SetDefault("http.drainWait", "0s")
	v.SetDefault("http.maxBodyBytes", 10<<20)
	v.SetDefault("http.trustedProxies", []string{"127.0.0.1", "::1", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"})
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")
	v.SetDefault("tracing.sampleRatio", 1.0)
}

// fillDefaults 为 map 类型的连接配置补默认值（viper 默认值无法作用于动态 map 键）
func (c *Config) fillDefaults() {
	for name, m := range c.MySQL {
		if m.Port == 0 {
			m.Port = 3306
		}
		if m.MaxOpenConns == 0 {
			m.MaxOpenConns = 30
		}
		if m.MaxIdleConns == 0 {
			m.MaxIdleConns = 10
		}
		if m.ConnMaxLifetime == 0 {
			m.ConnMaxLifetime = time.Hour
		}
		if m.ConnMaxIdleTime == 0 {
			m.ConnMaxIdleTime = 10 * time.Minute
		}
		if m.SlowThreshold == 0 {
			m.SlowThreshold = 500 * time.Millisecond
		}
		c.MySQL[name] = m
	}
}

func (c *Config) Validate() error {
	var errs []error
	add := func(format string, a ...any) { errs = append(errs, fmt.Errorf(format, a...)) }

	if c.App.Name == "" {
		add("app.name is required")
	}
	if c.HTTP.Port <= 0 || c.HTTP.Port > 65535 {
		add("http.port %d out of range", c.HTTP.Port)
	}
	switch c.Log.Level {
	case "debug", "info", "warn", "error":
	default:
		add("log.level %q must be one of debug|info|warn|error", c.Log.Level)
	}
	switch c.Log.Format {
	case "json", "text":
	default:
		add("log.format %q must be json or text", c.Log.Format)
	}
	durations := map[string]time.Duration{
		"http.readHeaderTimeout": c.HTTP.ReadHeaderTimeout,
		"http.readTimeout":       c.HTTP.ReadTimeout,
		"http.writeTimeout":      c.HTTP.WriteTimeout,
		"http.idleTimeout":       c.HTTP.IdleTimeout,
		"http.shutdownTimeout":   c.HTTP.ShutdownTimeout,
	}
	for name, m := range c.MySQL {
		if m.Host == "" || m.User == "" || m.Database == "" {
			add("mysql.%s: host, user and database are required", name)
		}
		durations["mysql."+name+".connMaxLifetime"] = m.ConnMaxLifetime
		durations["mysql."+name+".connMaxIdleTime"] = m.ConnMaxIdleTime
		durations["mysql."+name+".slowThreshold"] = m.SlowThreshold
	}
	for name, r := range c.Redis {
		if r.Addr == "" {
			add("redis.%s.addr is required", name)
		}
		durations["redis."+name+".connMaxIdleTime"] = r.ConnMaxIdleTime
		durations["redis."+name+".dialTimeout"] = r.DialTimeout
		durations["redis."+name+".readTimeout"] = r.ReadTimeout
		durations["redis."+name+".writeTimeout"] = r.WriteTimeout
	}
	// 时长必须带单位（如 8h、500ms）；裸数字会被解析为纳秒，这里拦截明显的误配
	for name, d := range durations {
		if d < 0 || (d > 0 && d < time.Millisecond) {
			add("%s=%s looks wrong, durations need a unit such as 500ms, 30s, 8h", name, d)
		}
	}
	if c.Tracing.Enable && c.Tracing.Endpoint == "" {
		add("tracing.endpoint is required when tracing.enable is true")
	}
	if c.Tracing.SampleRatio < 0 || c.Tracing.SampleRatio > 1 {
		add("tracing.sampleRatio must be within [0, 1]")
	}
	if c.RocketMQ.Enable && (len(c.RocketMQ.NameServers) == 0 || c.RocketMQ.GroupName == "") {
		add("rocketmq.nameServers and rocketmq.groupName are required when rocketmq.enable is true")
	}
	return errors.Join(errs...)
}
