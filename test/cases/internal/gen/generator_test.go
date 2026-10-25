package gen

import (
	"bytes"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// copyRepo 把仓库（不含 .git 等）复制到临时目录，生成器在副本上运行
func copyRepo(t *testing.T) string {
	t.Helper()
	src, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	dst := t.TempDir()
	err = filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		if d.IsDir() && (rel == ".git" || rel == ".idea" || rel == ".claude") {
			return filepath.SkipDir
		}
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(dst, rel), 0o755)
		}
		if !d.Type().IsRegular() || strings.HasSuffix(rel, ".log") {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dst, rel), b, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return dst
}

func newGen(t *testing.T) (*Generator, string) {
	t.Helper()
	dir := copyRepo(t)
	g, err := New(filepath.Join(dir, "app"))
	if err != nil {
		t.Fatal(err)
	}
	if g.Root != dir || g.ModulePath != "willshark" {
		t.Fatalf("root=%s module=%s", g.Root, g.ModulePath)
	}
	return g, dir
}

func apply(t *testing.T, g *Generator, o ModuleOptions, force bool) string {
	t.Helper()
	plan, err := g.Module(o)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := plan.Apply(&out, force, false); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestDependencyAutoInclude(t *testing.T) {
	g, dir := newGen(t)
	out := apply(t, g, ModuleOptions{Name: "order", With: []string{"router"}}, false)
	if !strings.Contains(out, "auto-included service") || !strings.Contains(out, "auto-included dto") {
		t.Fatalf("router should pull in service and dto:\n%s", out)
	}
	for _, f := range []string{"app/router/order.go", "app/service/order.go", "app/do/request/order.go", "app/do/response/order.go"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("%s not generated", f)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "app/controller/order.go")); err == nil {
		t.Error("controller should not be generated unless requested")
	}
}

func TestReuseExistingService(t *testing.T) {
	g, dir := newGen(t)
	apply(t, g, ModuleOptions{Name: "order"}, false)
	out := apply(t, g, ModuleOptions{Name: "order", With: []string{"job"}}, false)
	if strings.Contains(out, "app/service/order.go") {
		t.Fatalf("existing service must be reused, not regenerated:\n%s", out)
	}
	if job := read(t, filepath.Join(dir, "app/daemon_jobs/cron/order.go")); !strings.Contains(job, "Svc *service.Order") {
		t.Fatalf("job should depend on the existing service:\n%s", job)
	}
	if boot := read(t, filepath.Join(dir, bootstrapFile)); !strings.Contains(boot, "(&cron.OrderJob{Svc: orderSvc}).Register(sched)") {
		t.Fatal("job not wired with service")
	}
}

func TestConflictWritesNothing(t *testing.T) {
	g, dir := newGen(t)
	if err := os.WriteFile(filepath.Join(dir, "app/router/order.go"), []byte("package router\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := read(t, filepath.Join(dir, bootstrapFile))
	plan, err := g.Module(ModuleOptions{Name: "order"})
	if err != nil {
		t.Fatal(err)
	}
	err = plan.Apply(&bytes.Buffer{}, false, false)
	if err == nil || !strings.Contains(err.Error(), "app/router/order.go") {
		t.Fatalf("expected conflict error, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "app/service/order.go")); err == nil {
		t.Fatal("no file may be written when there is a conflict")
	}
	if read(t, filepath.Join(dir, bootstrapFile)) != before {
		t.Fatal("bootstrap must not change when there is a conflict")
	}
}

func TestForceIsIdempotentForWiring(t *testing.T) {
	g, dir := newGen(t)
	apply(t, g, ModuleOptions{Name: "order", With: []string{"all"}}, false)
	first := read(t, filepath.Join(dir, bootstrapFile))
	out := apply(t, g, ModuleOptions{Name: "order", With: []string{"all"}}, true)
	if !strings.Contains(out, "already wired") {
		t.Fatalf("second run should skip wiring:\n%s", out)
	}
	if read(t, filepath.Join(dir, bootstrapFile)) != first {
		t.Fatal("wiring must not be duplicated")
	}
	if strings.Count(first, "orderSvc :=") != 1 {
		t.Fatal("service wired more than once")
	}
}

func TestDryRunWritesNothing(t *testing.T) {
	g, dir := newGen(t)
	plan, err := g.Module(ModuleOptions{Name: "order", With: []string{"all"}})
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := plan.Apply(&out, false, true); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "[dry-run] + create  app/service/order.go") {
		t.Fatalf("dry-run output:\n%s", out.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "app/service/order.go")); err == nil {
		t.Fatal("dry-run must not write files")
	}
}

func TestMissingMarker(t *testing.T) {
	g, dir := newGen(t)
	p := filepath.Join(dir, routerBase)
	if err := os.WriteFile(p, []byte(strings.ReplaceAll(read(t, p), "// willshark:gen:router-register", "")), 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err := g.Module(ModuleOptions{Name: "order"})
	if err != nil {
		t.Fatal(err)
	}
	if err := plan.Apply(&bytes.Buffer{}, false, false); err == nil || !strings.Contains(err.Error(), "willshark:gen:router-register") {
		t.Fatalf("expected missing marker error, got %v", err)
	}
	// -no-wire 不依赖插入点
	plan, err = g.Module(ModuleOptions{Name: "order", NoWire: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := plan.Apply(&bytes.Buffer{}, false, false); err != nil {
		t.Fatal(err)
	}
}

func TestOptionErrors(t *testing.T) {
	g, dir := newGen(t)
	cases := map[string]ModuleOptions{
		"bad component": {Name: "order", With: []string{"routr"}},
		"bad name":      {Name: "1x"},
		"bad every":     {Name: "order", With: []string{"job"}, JobEvery: "soon"},
		"bad cron":      {Name: "order", With: []string{"job"}, JobCron: "every day"},
		"missing table": {Name: "nope", SQLFile: filepath.Join(dir, "sql/dynamic.sql")},
		"missing sql":   {Name: "task", SQLFile: filepath.Join(dir, "sql/none.sql")},
	}
	for name, o := range cases {
		if _, err := g.Module(o); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

func TestCustomActions(t *testing.T) {
	g, dir := newGen(t)
	apply(t, g, ModuleOptions{Name: "report", Actions: []string{"export", "daily-summary", "export"}}, false)
	r := read(t, filepath.Join(dir, "app/router/report.go"))
	if !strings.Contains(r, `grp.POST("/export", httpx.Handle(r.Svc.Export))`) || !strings.Contains(r, `grp.POST("/daily-summary", httpx.Handle(r.Svc.DailySummary))`) {
		t.Fatalf("router:\n%s", r)
	}
	if strings.Count(r, "/export") != 1 {
		t.Fatal("duplicate action should be ignored")
	}
	if s := read(t, filepath.Join(dir, "app/service/report.go")); !strings.Contains(s, "func (s *Report) DailySummary(ctx context.Context, req *request.DailySummaryReport) (*response.DailySummaryReport, error)") {
		t.Fatalf("service:\n%s", s)
	}
}

// TestGeneratedProjectCompiles 在仓库副本上执行多种组合，然后编译、vet 并运行 bootstrap 测试
func TestGeneratedProjectCompiles(t *testing.T) {
	if testing.Short() {
		t.Skip("compiles a full copy of the project")
	}
	g, dir := newGen(t)
	sqlFile := filepath.Join(dir, "sql/dynamic.sql")
	apply(t, g, ModuleOptions{Name: "order"}, false)
	apply(t, g, ModuleOptions{Name: "order", With: []string{"job"}, JobEvery: "90s"}, false)
	apply(t, g, ModuleOptions{Name: "report", With: []string{"all"}, Actions: []string{"list", "export"}, JobCron: "0 3 * * *"}, false)
	apply(t, g, ModuleOptions{Name: "task", SQLFile: sqlFile, With: []string{"all"}}, false)
	apply(t, g, ModuleOptions{Name: "contract", SQLFile: sqlFile, Table: "ann_order"}, false)
	apply(t, g, ModuleOptions{Name: "cleanup", With: []string{"job"}}, false)
	apply(t, g, ModuleOptions{Name: "audit", With: []string{"service"}}, false)
	apply(t, g, ModuleOptions{Name: "ping", With: []string{"dto"}, Actions: []string{"ping"}}, false)

	plan, err := g.Model(ModelOptions{SQLFile: sqlFile, Tables: []string{"ann_log_statistics", "ann_order_config"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := plan.Apply(&bytes.Buffer{}, false, false); err != nil {
		t.Fatal(err)
	}

	for _, args := range [][]string{{"run", "./test/run.go", "tidy"}, {"build", "./..."}, {"run", "./test/run.go", "vet", "./..."}, {"run", "./test/run.go", "-count=1", "./bootstrap/", "./app/..."}} {
		cmd := exec.Command("go", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("go %s failed: %v\n%s", strings.Join(args, " "), err, out)
		}
	}

	boot := strings.Join(strings.Fields(read(t, filepath.Join(dir, bootstrapFile))), " ") // 忽略 gofmt 对齐空格
	for _, want := range []string{
		"orderSvc := &service.Order{Pool: pool, Rds: rds}",
		"taskDao := &dao.Task{MysqlPool: pool}",
		"taskSvc := &service.Task{Dao: taskDao}",
		"reportCtl := &controller.Report{Svc: reportSvc}",
		"Report: &router.ReportRouter{Ctl: reportCtl},",
		"(&cron.OrderJob{Svc: orderSvc}).Register(sched)",
		"(&cron.CleanupJob{}).Register(sched)",
		"_ = auditSvc",
	} {
		if !strings.Contains(boot, want) {
			t.Errorf("bootstrap missing %q", want)
		}
	}
	if job := read(t, filepath.Join(dir, "app/daemon_jobs/cron/order.go")); !strings.Contains(job, `s.Every("order", 90*time.Second`) {
		t.Errorf("job schedule:\n%s", job)
	}
}

// 模块名生成的类型与已有类型重名（response.Empty）时整体中止，不写入任何文件
func TestRedeclaredIdentifierWritesNothing(t *testing.T) {
	g, dir := newGen(t)
	before := read(t, filepath.Join(dir, bootstrapFile))
	plan, err := g.Module(ModuleOptions{Name: "empty"})
	if err != nil {
		t.Fatal(err)
	}
	err = plan.Apply(&bytes.Buffer{}, false, false)
	if err == nil || !strings.Contains(err.Error(), "Empty already declared in app/do/response/common.go") {
		t.Fatalf("expected redeclaration error, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "app/service/empty.go")); err == nil {
		t.Fatal("no file may be written")
	}
	if read(t, filepath.Join(dir, bootstrapFile)) != before {
		t.Fatal("bootstrap must not change")
	}
}

func TestPlanFiles(t *testing.T) {
	g, _ := newGen(t)
	plan, err := g.Module(ModuleOptions{Name: "order", With: []string{"service"}})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(plan.Files(), ","); got != "app/do/request/order.go,app/do/response/order.go,app/service/order.go" {
		t.Fatalf("files = %s", got)
	}
}
