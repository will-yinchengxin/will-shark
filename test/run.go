// 测试文件集中在 test/cases，借助 Go 原生 overlay 在所属生产包中编译执行。
// 用法（仓库根目录）：go run ./test/run.go [-race] ./...
// vet：go run ./test/run.go vet ./...；整理依赖：go run ./test/run.go tidy
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func main() {
	if err := run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		return errors.New("cannot locate test runner source")
	}
	root := filepath.Dir(filepath.Dir(source))
	cases := filepath.Join(root, "test", "cases")
	replace := map[string]string{}
	err := filepath.WalkDir(cases, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(cases, path)
		if err != nil {
			return err
		}
		target := filepath.Join(root, rel)
		if !strings.HasSuffix(path, "_test.go") {
			return fmt.Errorf("unexpected file in test/cases: %s (fixtures belong in test/fixtures)", rel)
		}
		if _, err := os.Stat(target); !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("test must only exist in test/cases: %s", rel)
		}
		if info, err := os.Stat(filepath.Dir(target)); err != nil || !info.IsDir() {
			return fmt.Errorf("production package directory does not exist: %s", filepath.Dir(rel))
		}
		replace[target] = path
		return nil
	})
	if err != nil {
		return err
	}
	if len(replace) == 0 {
		return errors.New("no tests found in test/cases")
	}

	dir, err := os.MkdirTemp("", "willshark-overlay-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	b, err := json.Marshal(struct{ Replace map[string]string }{replace})
	if err != nil {
		return err
	}
	overlay := filepath.Join(dir, "overlay.json")
	if err := os.WriteFile(overlay, b, 0o600); err != nil {
		return err
	}

	args := os.Args[1:]
	command := "test"
	if len(args) > 0 && (args[0] == "test" || args[0] == "vet" || args[0] == "tidy") {
		command, args = args[0], args[1:]
	}
	if len(args) == 0 && command != "tidy" {
		args = []string{"./..."}
	}
	for _, arg := range args {
		if arg == "-overlay" || strings.HasPrefix(arg, "-overlay=") {
			return errors.New("the test runner manages -overlay; do not override it")
		}
	}
	fmt.Fprintf(os.Stderr, "willshark: %d centralized test files; go %s\n", len(replace), command)
	goArgs := []string{command, "-overlay=" + overlay}
	if command == "tidy" {
		goArgs = []string{"mod", "tidy", "-overlay=" + overlay}
	}
	cmd := exec.Command("go", append(goArgs, args...)...) // #nosec G204 G702 -- 本地测试入口有意转交开发者的 go test/vet/tidy 参数；可执行程序固定为 go，未经过 shell
	cmd.Dir = root
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}
