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
		"redis missing addr":   {"redis:\n  main:\n    db: 1\n", "redis.main.addr is required"},
		"tracing endpoint":     {"tracing:\n  enable: true\n", "tracing.endpoint is required"},
		"bad log level":        {"log:\n  level: verbose\n", "log.level"},
		"bad sample ratio":     {"tracing:\n  sampleRatio: 2\n", "sampleRatio"},
		"rocketmq":             {"rocketmq:\n  enable: true\n", "rocketmq.nameServers"},
		"bad port":             {"http:\n  port: 70000\n", "http.port"},
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
