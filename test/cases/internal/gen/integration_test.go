package gen

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// 在仓库副本生成模块，使用真实 MySQL 验证 HTTP CRUD，避免仅编译通过而运行时不可用。
func TestIntegrationGeneratedCRUD(t *testing.T) {
	if os.Getenv("WS_INTEGRATION") == "" {
		t.Skip("set WS_INTEGRATION=1 to run generated CRUD against docker compose")
	}
	g, dir := newGen(t)
	sqlFile := filepath.Join(dir, "test", "fixtures", "crud.sql")
	apply(t, g, ModuleOptions{Name: "ticketprobe", SQLFile: sqlFile, Table: "ws_gen_crud_probe"}, false)
	apply(t, g, ModuleOptions{Name: "intprobe", SQLFile: sqlFile, Table: "ws_gen_int_probe"}, false)
	fixture, err := os.ReadFile(filepath.Join(dir, "test", "fixtures", "generated_crud_test.go.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "test", "cases", "bootstrap", "generated_crud_test.go"), fixture, 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", "./test/run.go", "-race", "-count=1", "-run", "^TestIntegrationGeneratedCRUD$", "./bootstrap")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generated CRUD: %v\n%s", err, out)
	}
}
