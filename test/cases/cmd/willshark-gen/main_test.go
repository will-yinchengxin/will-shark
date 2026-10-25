package main

import (
	"bytes"
	"errors"
	"flag"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestCLI(t *testing.T) {
	var out, errOut bytes.Buffer
	if err := run(nil, &out, &errOut); !errors.Is(err, flag.ErrHelp) || !strings.Contains(errOut.String(), "用法") {
		t.Fatalf("no args: %v", err)
	}
	if err := run([]string{"bogus"}, &out, &errOut); err == nil {
		t.Fatal("unknown command should fail")
	}
	if err := run([]string{"module"}, &out, &errOut); err == nil {
		t.Fatal("module without name should fail")
	}
	if err := run([]string{"model"}, &out, &errOut); err == nil {
		t.Fatal("model without -sql should fail")
	}
}

// 名称可以放在参数前或参数后；dry-run 不写任何文件
func TestModuleDryRunBothArgOrders(t *testing.T) {
	for _, args := range [][]string{
		{"module", "zz_cli_probe", "-with", "all", "-dry-run", "-dir", "../.."},
		{"module", "-with", "all", "-dry-run", "-dir", "../..", "zz_cli_probe"},
	} {
		var out bytes.Buffer
		if err := run(args, &out, &bytes.Buffer{}); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		for _, want := range []string{"[dry-run] + create  app/service/zz_cli_probe.go", "[dry-run] + create  app/daemon_jobs/cron/zz_cli_probe.go", "[dry-run] ~ update  bootstrap/app.go"} {
			if !strings.Contains(out.String(), want) {
				t.Fatalf("%v: missing %q in\n%s", args, want, out.String())
			}
		}
	}
	if _, err := os.Stat("../../app/service/zz_cli_probe.go"); err == nil {
		t.Fatal("dry-run wrote a file")
	}
}

func TestModelDryRun(t *testing.T) {
	var out bytes.Buffer
	if err := run([]string{"model", "-sql", "../../sql/dynamic.sql", "-tables", "task", "-dry-run", "-dir", "../.."}, &out, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"[dry-run] + create  app/modules/mysql/entity/task.go", "[dry-run] + create  app/modules/mysql/dao/task.go"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("missing %q in\n%s", want, out.String())
		}
	}
	if _, err := os.Stat("../../app/modules/mysql/entity/task.go"); err == nil {
		t.Fatal("dry-run wrote a file")
	}
}

// main 在命令出错时以退出码 1 结束，-h 不算错误
func TestMainExitCode(t *testing.T) {
	if args := os.Getenv("WS_GEN_SUBPROCESS"); args != "" {
		os.Args = append([]string{"willshark-gen"}, strings.Fields(args)...)
		main()
		return
	}
	run := func(args string) (int, string) {
		cmd := exec.Command(os.Args[0], "-test.run=^TestMainExitCode$")
		cmd.Env = append(os.Environ(), "WS_GEN_SUBPROCESS="+args)
		out, err := cmd.CombinedOutput()
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode(), string(out)
		}
		return 0, string(out)
	}
	if code, out := run("bogus"); code != 1 || !strings.Contains(out, `unknown command "bogus"`) {
		t.Fatalf("bogus: code=%d out=%s", code, out)
	}
	if code, _ := run("help"); code != 0 {
		t.Fatalf("help: code=%d", code)
	}
}
