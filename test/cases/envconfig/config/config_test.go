package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "test_config.yaml")
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

const minimal = `
mysql:
  main:
    host: db.local
    user: u
    password: from-file
    database: d
redis:
  main:
    addr: 127.0.0.1:6379
`

func TestLoadDefaults(t *testing.T) {
	c, err := LoadFile("test", writeConfig(t, minimal))
	if err != nil {
		t.Fatal(err)
	}
	if c.App.Env != "test" || c.App.Name == "" || c.HTTP.Port != 8899 {
		t.Fatalf("app/http defaults wrong: %+v %+v", c.App, c.HTTP)
	}
	if c.HTTP.ReadHeaderTimeout != 5*time.Second || c.HTTP.ShutdownTimeout != 25*time.Second || c.HTTP.MaxBodyBytes != 10<<20 {
		t.Fatalf("http defaults wrong: %+v", c.HTTP)
	}
	if c.Log.Level != "info" || c.Log.Format != "json" || c.Tracing.SampleRatio != 1 {
		t.Fatalf("log/tracing defaults wrong: %+v %+v", c.Log, c.Tracing)
	}
	m := c.MySQL["main"]
	if m.Port != 3306 || m.MaxOpenConns != 30 || m.ConnMaxLifetime != time.Hour || m.SlowThreshold != 500*time.Millisecond {
		t.Fatalf("mysql defaults wrong: %+v", m)
	}
}

func TestEnvOverride(t *testing.T) {
	t.Setenv("WS_MYSQL_MAIN_PASSWORD", "from-env")
	t.Setenv("WS_HTTP_PORT", "9000")
	t.Setenv("WS_LOG_LEVEL", "debug")
	c, err := LoadFile("test", writeConfig(t, minimal))
	if err != nil {
		t.Fatal(err)
	}
	if got := c.MySQL["main"].Password; got != "from-env" {
		t.Fatalf("password = %q, want env override", got)
	}
	if c.HTTP.Port != 9000 || c.Log.Level != "debug" {
		t.Fatalf("env override failed: port=%d level=%s", c.HTTP.Port, c.Log.Level)
	}
}

func TestDurationsWithUnits(t *testing.T) {
	c, err := LoadFile("test", writeConfig(t, minimal+`
http:
  drainWait: 5s
  writeTimeout: 2m
`))
	if err != nil {
		t.Fatal(err)
	}
	if c.HTTP.DrainWait != 5*time.Second || c.HTTP.WriteTimeout != 2*time.Minute {
		t.Fatalf("durations: %+v", c.HTTP)
	}
}

func TestValidation(t *testing.T) {
	cases := map[string]struct {
		yaml string
		want string
	}{
		"bare number duration": {minimal + "http:\n  readTimeout: 30\n", "durations need a unit"},
		"mysql missing host":   {"mysql:\n  main:\n    user: u\n    database: d\n", "mysql.main: host, user and database are required"},
		"redis missing addr":   {"redis:\n  main:\n    db: 1\n", "redis.main.addr is required in single mode"},
		"redis cluster addrs":  {"redis:\n  main:\n    mode: cluster\n", "addrs is required in cluster mode"},
		"redis cluster db":     {"redis:\n  main:\n    mode: cluster\n    addrs: [a:1]\n    db: 2\n", "db must be 0"},
		"redis sentinel":       {"redis:\n  main:\n    mode: sentinel\n    addrs: [a:1]\n", "masterName"},
		"redis bad mode":       {"redis:\n  main:\n    mode: x\n    addr: a:1\n", "mode \"x\""},
		"replica host":         {"mysql:\n  m:\n    host: h\n    user: u\n    database: d\n    replicas:\n      - port: 3307\n", "replicas[0].host is required"},
		"bad policy":           {"mysql:\n  m:\n    host: h\n    user: u\n    database: d\n    policy: x\n", "policy"},
		"tracing endpoint":     {"tracing:\n  enable: true\n", "tracing.endpoint is required"},
		"bad log level":        {"log:\n  level: verbose\n", "log.level"},
		"bad sample ratio":     {"tracing:\n  sampleRatio: 2\n", "sampleRatio"},
		"rocketmq":             {"rocketmq:\n  enable: true\n", "rocketmq.nameServers"},
		"bad port":             {"http:\n  port: 70000\n", "http.port"},
		"no header timeout":    {"http:\n  readHeaderTimeout: 0s\n", "http.readHeaderTimeout must be positive"},
		"no shutdown timeout":  {"http:\n  shutdownTimeout: 0s\n", "http.shutdownTimeout must be positive"},
		"no body limit":        {"http:\n  maxBodyBytes: -1\n", "http.maxBodyBytes must be positive"},
		"negative drain":       {"http:\n  drainWait: -5s\n", "http.drainWait"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := LoadFile("test", writeConfig(t, tc.yaml))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want contains %q", err, tc.want)
			}
		})
	}
}

func TestLoadMissingFile(t *testing.T) {
	if _, err := LoadFile("test", filepath.Join(t.TempDir(), "nope.yaml")); err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveEnv(t *testing.T) {
	t.Setenv(EnvAppEnv, "")
	if got := ResolveEnv(nil); got != "dev" {
		t.Fatalf("default = %s", got)
	}
	if got := ResolveEnv([]string{"pro"}); got != "pro" {
		t.Fatalf("arg = %s", got)
	}
	t.Setenv(EnvAppEnv, "uat")
	if got := ResolveEnv([]string{"pro"}); got != "uat" {
		t.Fatalf("APP_ENV should win, got %s", got)
	}
}

// 仓库内的配置文件必须能通过校验（pro/uat 需注入主机地址）
func TestRepoConfigFiles(t *testing.T) {
	t.Setenv(EnvConfigDir, "..")
	if _, err := Load("dev"); err != nil {
		t.Fatalf("dev config: %v", err)
	}
	for _, env := range []string{"uat", "pro"} {
		if _, err := Load(env); err == nil {
			t.Fatalf("%s config without injected hosts should fail validation", env)
		}
		t.Setenv("WS_MYSQL_WILLSHARK_HOST", "db.internal")
		t.Setenv("WS_REDIS_WILLSHARK_ADDR", "redis.internal:6379")
		c, err := Load(env)
		if err != nil {
			t.Fatalf("%s config with env: %v", env, err)
		}
		if c.App.Swagger || c.HTTP.DrainWait != 5*time.Second {
			t.Fatalf("%s should disable swagger and set drainWait: %+v", env, c.HTTP)
		}
		t.Setenv("WS_MYSQL_WILLSHARK_HOST", "")
		t.Setenv("WS_REDIS_WILLSHARK_ADDR", "")
	}
}

func TestRealTopic(t *testing.T) {
	r := RocketMQ{Topics: map[string]string{"a": "real-a"}}
	if r.RealTopic("a") != "real-a" || r.RealTopic("b") != "b" {
		t.Fatal("RealTopic mapping wrong")
	}
}

func TestMySQLNodesInheritPrimary(t *testing.T) {
	c, err := LoadFile("test", writeConfig(t, `
mysql:
  main:
    host: primary
    port: 3306
    user: u
    password: p
    database: d
    policy: round_robin
    sources:
      - host: primary2
    replicas:
      - host: replica1
        port: 3307
      - host: replica2
        user: ro
        password: rop
`))
	if err != nil {
		t.Fatal(err)
	}
	m := c.MySQL["main"]
	if m.Policy != "round_robin" || len(m.Sources) != 1 || len(m.Replicas) != 2 {
		t.Fatalf("%+v", m)
	}
	if s := m.Sources[0]; s.Port != 3306 || s.User != "u" || s.Password != "p" || s.Database != "d" {
		t.Fatalf("source should inherit primary: %+v", s)
	}
	if r := m.Replicas[0]; r.Port != 3307 || r.User != "u" {
		t.Fatalf("replica1: %+v", r)
	}
	if r := m.Replicas[1]; r.User != "ro" || r.Password != "rop" || r.Port != 3306 {
		t.Fatalf("replica2 should keep own credentials: %+v", r)
	}
}

func TestRedisModesAndEnvSlice(t *testing.T) {
	t.Setenv("WS_REDIS_MAIN_ADDRS", "n1:6379,n2:6379,n3:6379")
	c, err := LoadFile("test", writeConfig(t, `
redis:
  main:
    mode: cluster
    addrs: [placeholder:6379]
  single:
    addr: 127.0.0.1:6379
`))
	if err != nil {
		t.Fatal(err)
	}
	if got := c.Redis["main"].Addrs; len(got) != 3 || got[2] != "n3:6379" {
		t.Fatalf("addrs env override = %v", got)
	}
	if c.Redis["single"].Mode != RedisModeSingle {
		t.Fatalf("default mode = %q", c.Redis["single"].Mode)
	}
}

// 配置文件中没写的键也能被环境变量覆盖（例如 pro 模板里没有 redis.mode / addrs）
func TestEnvOverrideKeysMissingFromFile(t *testing.T) {
	t.Setenv("WS_HTTP_ALWAYSOK", "true")
	t.Setenv("WS_TRACING_ENABLE", "true")
	t.Setenv("WS_TRACING_ENDPOINT", "otel:4318")
	t.Setenv("WS_REDIS_MAIN_MODE", "cluster")
	t.Setenv("WS_REDIS_MAIN_ADDRS", "n1:6379,n2:6379")
	t.Setenv("WS_MYSQL_MAIN_SLOWTHRESHOLD", "1s")
	c, err := LoadFile("test", writeConfig(t, minimal))
	if err != nil {
		t.Fatal(err)
	}
	r := c.Redis["main"]
	if !c.HTTP.AlwaysOK || !c.Tracing.Enable || c.Tracing.Endpoint != "otel:4318" ||
		r.Mode != RedisModeCluster || strings.Join(r.Addrs, ",") != "n1:6379,n2:6379" ||
		c.MySQL["main"].SlowThreshold != time.Second || c.MySQL["main"].Host != "db.local" {
		t.Fatalf("env not applied: http=%+v tracing=%+v redis=%+v", c.HTTP, c.Tracing, r)
	}
}

func TestLoadDefaultDir(t *testing.T) {
	t.Setenv(EnvConfigDir, "")
	wd, _ := os.Getwd()
	t.Chdir(filepath.Join(wd, "..", ".."))
	if _, err := Load("dev"); err != nil {
		t.Fatalf("default dir envconfig: %v", err)
	}
}
