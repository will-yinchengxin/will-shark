package main

import (
	"errors"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// run 的启动链路：配置错误与依赖不可达都应返回错误（main 据此以非 0 退出），而不是 panic
func TestRunFailsFast(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("WS_CONFIG_DIR", dir)
	t.Setenv("APP_ENV", "test")

	if err := run(); err == nil || !strings.Contains(err.Error(), "read config") {
		t.Fatalf("missing config: %v", err)
	}

	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()
	cfg := "log:\n  format: json\nmysql:\n  willshark:\n    host: 127.0.0.1\n    port: " + strconv.Itoa(port) +
		"\n    user: u\n    database: d\nredis:\n  willshark:\n    addr: 127.0.0.1:" + strconv.Itoa(port) + "\n"
	if err := os.WriteFile(filepath.Join(dir, "test_config.yaml"), []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := run(); err == nil || !strings.Contains(err.Error(), "init mysql willshark") {
		t.Fatalf("unreachable mysql: %v", err)
	}
}

// main 在 run 失败时以退出码 1 结束：以子进程方式运行本测试二进制验证
func TestMainExitsNonZeroOnError(t *testing.T) {
	if os.Getenv("WS_MAIN_SUBPROCESS") == "1" {
		os.Args = []string{"willshark"}
		main()
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestMainExitsNonZeroOnError$")
	cmd.Env = append(os.Environ(), "WS_MAIN_SUBPROCESS=1", "WS_CONFIG_DIR="+t.TempDir(), "APP_ENV=test")
	out, err := cmd.CombinedOutput()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 || !strings.Contains(string(out), "exit with error") {
		t.Fatalf("err=%v out=%s", err, out)
	}
}
