package gen

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/swaggo/swag"
)

func TestDefaultModuleUsesController(t *testing.T) {
	g, dir := newGen(t)
	apply(t, g, ModuleOptions{Name: "order"}, false)
	r := read(t, filepath.Join(dir, "app/router/order.go"))
	if !strings.Contains(r, "Ctl *controller.Order") || !strings.Contains(r, `grp.POST("", r.Ctl.Create)`) || strings.Contains(r, "httpx.Handle") || strings.Contains(r, "/app/service") {
		t.Fatalf("router must delegate HTTP handling to controller:\n%s", r)
	}
	c := read(t, filepath.Join(dir, "app/controller/order.go"))
	if !strings.Contains(c, "httpx.Bind(ctx, &req)") || !strings.Contains(c, "c.Svc.Create(ctx.Request.Context(), &req)") || !strings.Contains(c, "@Router") || !strings.Contains(c, "httpx.OK(ctx, resp)") {
		t.Fatalf("controller must own input/output and Swagger:\n%s", c)
	}
	s := read(t, filepath.Join(dir, "app/service/order.go"))
	if strings.Contains(s, "@Router") || strings.Contains(s, "@Param") || strings.Contains(s, "gin.Context") {
		t.Fatalf("service must not own HTTP metadata:\n%s", s)
	}
}

func checkGeneratedSwagger(t *testing.T, dir string) {
	t.Helper()
	p := swag.New(swag.SetStrict(true))
	p.ParseInternal = true
	if err := p.ParseAPI(dir, "main.go", 100); err != nil {
		t.Fatalf("Swagger from generated controllers: %v", err)
	}
	spec := p.GetSwagger()
	for _, module := range []string{"order", "task", "contract", "ledger"} {
		base, ok := spec.Paths.Paths["/v1/"+module]
		if !ok || base.Get == nil || base.Post == nil {
			t.Fatalf("%s list/create missing from Swagger", module)
		}
		item, ok := spec.Paths.Paths["/v1/"+module+"/{id}"]
		if !ok || item.Get == nil || item.Put == nil || item.Delete == nil {
			t.Fatalf("%s get/update/delete missing from Swagger", module)
		}
	}
	for _, name := range []string{"request.CreateOrder", "response.Order", "response.OrderList", "httpx.Response"} {
		if _, ok := spec.Definitions[name]; !ok {
			t.Errorf("Swagger DTO missing: %s", name)
		}
	}
	if _, ok := spec.Paths.Paths["/v1/audit"]; ok {
		t.Error("service-only generation must not publish HTTP operations")
	}
}

func TestRouterReusesExistingController(t *testing.T) {
	g, dir := newGen(t)
	apply(t, g, ModuleOptions{Name: "order", With: []string{CompController}}, false)
	before := read(t, filepath.Join(dir, "app/controller/order.go"))
	out := apply(t, g, ModuleOptions{Name: "order", With: []string{CompRouter}}, false)
	if strings.Contains(out, "+ create  app/controller/") || read(t, filepath.Join(dir, "app/controller/order.go")) != before {
		t.Fatal("existing controller should be reused")
	}
	r := read(t, filepath.Join(dir, "app/router/order.go"))
	if !strings.Contains(r, "r.Ctl.Create") {
		t.Fatalf("router bypassed controller:\n%s", r)
	}
}

func TestLegacyServiceRouterRequiresExplicitMigration(t *testing.T) {
	g, dir := newGen(t)
	p := filepath.Join(dir, bootstrapFile)
	before := strings.Replace(read(t, p), "// willshark:gen:router", "Order: &router.OrderRouter{Svc: orderSvc},\n\t\t// willshark:gen:router", 1)
	if err := os.WriteFile(p, []byte(before), 0o600); err != nil {
		t.Fatal(err)
	}
	plan, err := g.Module(ModuleOptions{Name: "order"})
	if err == nil {
		err = plan.Apply(&bytes.Buffer{}, true, false)
	}
	if err == nil || !strings.Contains(err.Error(), "controller") {
		t.Fatalf("legacy wiring should be rejected before duplicate entries are written: %v", err)
	}
	if read(t, p) != before {
		t.Fatal("legacy migration error must not change bootstrap")
	}
}

func TestMultilineLegacyRouterRequiresExplicitMigration(t *testing.T) {
	g, dir := newGen(t)
	p := filepath.Join(dir, bootstrapFile)
	before := strings.Replace(read(t, p), "// willshark:gen:router", "Order: &router.OrderRouter{\nSvc: orderSvc,\n},\n// willshark:gen:router", 1)
	if err := os.WriteFile(p, []byte(before), 0o600); err != nil {
		t.Fatal(err)
	}
	plan, err := g.Module(ModuleOptions{Name: "order"})
	if err == nil {
		err = plan.Apply(&bytes.Buffer{}, true, false)
	}
	if err == nil || !strings.Contains(err.Error(), "controller") {
		t.Fatalf("multiline legacy wiring must fail safely: %v", err)
	}
	if read(t, p) != before {
		t.Fatal("bootstrap changed on rejected migration")
	}
}

func TestLegacyRouterAllowsNoWireMigration(t *testing.T) {
	g, dir := newGen(t)
	p := filepath.Join(dir, bootstrapFile)
	before := strings.Replace(read(t, p), "// willshark:gen:router", "Order: &router.OrderRouter{Svc: orderSvc},\n// willshark:gen:router", 1)
	if err := os.WriteFile(p, []byte(before), 0o600); err != nil {
		t.Fatal(err)
	}
	apply(t, g, ModuleOptions{Name: "order", NoWire: true}, true)
	if read(t, p) != before {
		t.Fatal("-no-wire changed bootstrap")
	}
	if !strings.Contains(read(t, filepath.Join(dir, "app/router/order.go")), "r.Ctl.Create") {
		t.Fatal("manual migration did not generate controller-based route")
	}
}

func TestAliasedLegacyRouterRequiresExplicitMigration(t *testing.T) {
	g, dir := newGen(t)
	p := filepath.Join(dir, bootstrapFile)
	before := strings.ReplaceAll(read(t, p), "router.", "routes.")
	before = strings.Replace(before, `"willshark/app/router"`, `routes "willshark/app/router"`, 1)
	before = strings.Replace(before, "// willshark:gen:router", "Order: &routes.OrderRouter{\nSvc: orderSvc,\n},\n// willshark:gen:router", 1)
	if err := os.WriteFile(p, []byte(before), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := g.Module(ModuleOptions{Name: "order"}); err == nil || !strings.Contains(err.Error(), "controller") {
		t.Fatalf("aliased legacy wiring must be rejected: %v", err)
	}
	if read(t, p) != before {
		t.Fatal("bootstrap changed on rejected migration")
	}
}
