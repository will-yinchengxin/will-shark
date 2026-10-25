// Package gen willshark 代码生成器：按需组合生成 router / controller / service / dto / job，
// 以及从 CREATE TABLE 生成 entity / dao 与完整 CRUD，并自动接入 bootstrap 与路由。
package gen

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"text/template"
	"time"

	"golang.org/x/tools/imports"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

var tmpl = template.Must(template.New("").Funcs(template.FuncMap{
	"lower": strings.ToLower,
	"wrap": func(r *data, a Action) any {
		return struct {
			R *data
			A Action
		}{r, a}
	},
	"swagType": func(goType string) string {
		if strings.Contains(goType, "int") {
			return "integer"
		}
		return "string"
	},
}).ParseFS(templateFS, "templates/*.tmpl"))

// 组件
const (
	CompRouter     = "router"
	CompController = "controller"
	CompService    = "service"
	CompDTO        = "dto"
	CompJob        = "job"
)

var allComponents = []string{CompRouter, CompController, CompService, CompDTO, CompJob}

// DefaultComponents 未指定 -with 时生成的组件
var DefaultComponents = []string{CompRouter, CompService, CompDTO}

// DefaultActions 未指定 -actions 时生成的接口
var DefaultActions = []string{"list", "get", "create", "update", "delete"}

// ModuleOptions module 命令参数
type ModuleOptions struct {
	Name     string
	With     []string // 组件；会自动补齐依赖（router/controller → service → dto）
	Actions  []string // list,get,create,update,delete 或任意自定义动作（POST /<action>）
	SQLFile  string   // 指定后从表结构生成 entity/dao 与 CRUD 实现
	Table    string   // SQL 文件中的表名，默认与 Name 相同
	JobEvery string   // 定时任务间隔，如 5m
	JobCron  string   // 定时任务 cron 表达式，优先于 JobEvery
	NoWire   bool     // 不修改 bootstrap 与 router/base.go
}

// ModelOptions model 命令参数
type ModelOptions struct {
	SQLFile string
	Tables  []string // 为空则生成文件中的全部表
}

// Action 一个接口
type Action struct {
	Kind     string // list get create update delete custom
	Method   string // service 方法名
	HTTP     string // GET POST PUT DELETE
	Path     string // 相对路由组的路径
	SwagPath string
	Req      string
	Resp     string
	Summary  string
	HasID    bool
	HasBody  bool
}

type data struct {
	Module      string
	N           Name
	Summary     string
	Actions     []Action
	Table       *TableData
	IDType      string
	NeedItem    bool
	HasList     bool
	UseCtl      bool
	JobSvc      bool
	JobSchedule string
}

type modelData struct {
	Module string
	Type   string
	IDType string
	PKCol  string
	Table  *TableData
}

// Generator 绑定到一个 Go 项目根目录（包含 go.mod）
type Generator struct {
	Root       string
	ModulePath string // go.mod 中的 module 路径
}

// New 从 dir 向上查找 go.mod
func New(dir string) (*Generator, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	for d := dir; ; d = filepath.Dir(d) {
		b, err := os.ReadFile(filepath.Join(d, "go.mod"))
		if err == nil {
			m := regexp.MustCompile(`(?m)^module\s+(\S+)`).FindSubmatch(b)
			if m == nil {
				return nil, fmt.Errorf("%s/go.mod: module path not found", d)
			}
			return &Generator{Root: d, ModulePath: string(m[1])}, nil
		}
		if filepath.Dir(d) == d {
			return nil, fmt.Errorf("go.mod not found from %s", dir)
		}
	}
}

func (g *Generator) path(parts ...string) string {
	return filepath.Join(append([]string{g.Root}, parts...)...)
}

func (g *Generator) exists(rel string) bool {
	_, err := os.Stat(g.path(rel))
	return err == nil
}

func (g *Generator) bootstrapHas(line string) bool {
	b, err := os.ReadFile(g.path(bootstrapFile))
	return err == nil && bytes.Contains(b, []byte(line))
}

const (
	bootstrapFile = "bootstrap/app.go"
	routerBase    = "app/router/base.go"
)

// Module 生成模块
func (g *Generator) Module(o ModuleOptions) (*Plan, error) {
	n, err := NewName(o.Name)
	if err != nil {
		return nil, err
	}
	with, err := normalizeComponents(o.With)
	if err != nil {
		return nil, err
	}
	actions := o.Actions
	if len(actions) == 0 {
		actions = DefaultActions
	}
	plan := &Plan{gen: g}

	var table *TableData
	if o.SQLFile != "" {
		tableName := o.Table
		if tableName == "" {
			tableName = n.Snake
		}
		t, err := g.loadTable(o.SQLFile, tableName)
		if err != nil {
			return nil, err
		}
		if table, err = newTableData(t); err != nil {
			return nil, err
		}
	}

	f := files(n)
	// 已存在且已接入的组件可直接复用
	svcVar, ctlVar := n.Camel+"Svc", n.Camel+"Ctl"
	svcReady := g.exists(f.service) && g.bootstrapHas(svcVar+" :=")
	ctlReady := g.exists(f.controller) && g.bootstrapHas(ctlVar+" :=")
	dtoReady := g.exists(f.request)

	// 依赖补齐：router/controller 需要 service，service/controller 需要 dto
	need := func(c string) {
		if !slices.Contains(with, c) {
			with = append(with, c)
			plan.note("auto-included %s (required by other components)", c)
		}
	}
	if (slices.Contains(with, CompRouter) || slices.Contains(with, CompController)) && !svcReady {
		need(CompService)
	}
	if (slices.Contains(with, CompService) || slices.Contains(with, CompController)) && !dtoReady {
		need(CompDTO)
	}
	has := func(c string) bool { return slices.Contains(with, c) }

	d := &data{
		Module:  g.ModulePath,
		N:       n,
		Summary: n.Pascal,
		Table:   table,
		IDType:  "int64",
		UseCtl:  has(CompController) || ctlReady,
		JobSvc:  has(CompService) || svcReady,
	}
	if table != nil {
		d.IDType = table.PK.BaseType
		if table.Comment != "" {
			d.Summary = table.Comment
		}
	}
	if d.Actions, err = buildActions(n, actions, d.Summary); err != nil {
		return nil, err
	}
	for _, a := range d.Actions {
		switch a.Kind {
		case "list":
			d.HasList, d.NeedItem = true, true
		case "get", "create", "update":
			d.NeedItem = true
		}
	}
	if has(CompJob) {
		if d.JobSchedule, err = jobSchedule(n, o.JobEvery, o.JobCron); err != nil {
			return nil, err
		}
	}

	render := func(rel, name string, v any) error {
		src, err := renderGo(rel, name, v)
		if err != nil {
			return err
		}
		plan.create(rel, src)
		return nil
	}
	if table != nil {
		md := &modelData{Module: g.ModulePath, Type: n.Pascal, IDType: table.PK.BaseType, PKCol: table.PK.Column, Table: table}
		if err := render(f.entity, "entity.tmpl", md); err != nil {
			return nil, err
		}
		if err := render(f.dao, "dao.tmpl", md); err != nil {
			return nil, err
		}
	}
	steps := []struct {
		comp, rel, tmpl string
	}{
		{CompDTO, f.request, "request.tmpl"},
		{CompDTO, f.response, "response.tmpl"},
		{CompService, f.service, "service.tmpl"},
		{CompController, f.controller, "controller.tmpl"},
		{CompRouter, f.router, "router.tmpl"},
		{CompJob, f.job, "job.tmpl"},
	}
	for _, s := range steps {
		if has(s.comp) {
			if err := render(s.rel, s.tmpl, d); err != nil {
				return nil, err
			}
		}
	}

	if o.NoWire {
		plan.note("-no-wire: remember to wire the new components in bootstrap/app.go and app/router/base.go")
		return plan, nil
	}
	g.wire(plan, n, d, has, table != nil)
	return plan, nil
}

// wire 在 bootstrap.wire 与 router/base.go 的插入点添加接入代码
func (g *Generator) wire(plan *Plan, n Name, d *data, has func(string) bool, withTable bool) {
	svcVar, ctlVar, daoVar := n.Camel+"Svc", n.Camel+"Ctl", n.Camel+"Dao"
	consumed := false
	if withTable && has(CompService) {
		plan.insert(bootstrapFile, "willshark:gen:dao", fmt.Sprintf("%s := &dao.%s{MysqlPool: pool}", daoVar, n.Pascal))
	}
	if has(CompService) {
		if withTable {
			plan.insert(bootstrapFile, "willshark:gen:service", fmt.Sprintf("%s := &service.%s{Dao: %s}", svcVar, n.Pascal, daoVar))
		} else {
			plan.insert(bootstrapFile, "willshark:gen:service", fmt.Sprintf("%s := &service.%s{Pool: pool, Rds: rds}", svcVar, n.Pascal))
		}
	}
	if has(CompController) {
		plan.insert(bootstrapFile, "willshark:gen:controller", fmt.Sprintf("%s := &controller.%s{Svc: %s}", ctlVar, n.Pascal, svcVar))
		plan.addImport(bootstrapFile, g.ModulePath+"/app/controller")
		consumed = true
	}
	if has(CompRouter) {
		field := fmt.Sprintf("Svc: %s", svcVar)
		if d.UseCtl {
			field = fmt.Sprintf("Ctl: %s", ctlVar)
		}
		plan.insert(bootstrapFile, "willshark:gen:router", fmt.Sprintf("%s: &router.%sRouter{%s},", n.Pascal, n.Pascal, field))
		plan.insert(routerBase, "willshark:gen:router-field", fmt.Sprintf("%s *%sRouter", n.Pascal, n.Pascal))
		plan.insert(routerBase, "willshark:gen:router-register", fmt.Sprintf("r.%s.Register(v1)", n.Pascal))
		consumed = true
	}
	if has(CompJob) {
		if d.JobSvc {
			plan.insert(bootstrapFile, "willshark:gen:job", fmt.Sprintf("(&cron.%sJob{Svc: %s}).Register(sched)", n.Pascal, svcVar))
		} else {
			plan.insert(bootstrapFile, "willshark:gen:job", fmt.Sprintf("(&cron.%sJob{}).Register(sched)", n.Pascal))
		}
		consumed = consumed || d.JobSvc
	}
	if has(CompService) && !consumed {
		// 暂无使用方时避免 "declared and not used"
		plan.insert(bootstrapFile, "willshark:gen:service", fmt.Sprintf("_ = %s // TODO: 注入到 router 或定时任务", svcVar))
	}
}

// Model 只生成 entity 与 dao
func (g *Generator) Model(o ModelOptions) (*Plan, error) {
	src, err := os.ReadFile(o.SQLFile)
	if err != nil {
		return nil, err
	}
	tables, err := ParseSQL(string(src))
	if err != nil {
		return nil, err
	}
	plan := &Plan{gen: g}
	found := map[string]bool{}
	for _, t := range tables {
		if len(o.Tables) > 0 && !slices.Contains(o.Tables, t.Name) {
			continue
		}
		found[t.Name] = true
		n, err := NewName(t.Name)
		if err != nil {
			return nil, err
		}
		td, err := newTableData(t)
		md := &modelData{Module: g.ModulePath, Type: n.Pascal, Table: td}
		f := files(n)
		if r, rerr := renderGo(f.entity, "entity.tmpl", md); rerr != nil {
			return nil, rerr
		} else {
			plan.create(f.entity, r)
		}
		if err != nil {
			plan.note("table %s: no single-column primary key, dao skipped", t.Name)
			continue
		}
		md.IDType, md.PKCol = td.PK.BaseType, td.PK.Column
		r, err := renderGo(f.dao, "dao.tmpl", md)
		if err != nil {
			return nil, err
		}
		plan.create(f.dao, r)
	}
	for _, want := range o.Tables {
		if !found[want] {
			return nil, fmt.Errorf("table %q not found in %s", want, o.SQLFile)
		}
	}
	plan.note("entity/dao are not wired automatically; create the dao in bootstrap.wire, e.g. xxxDao := &dao.Xxx{MysqlPool: pool}")
	return plan, nil
}

func (g *Generator) loadTable(file, name string) (*Table, error) {
	src, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	tables, err := ParseSQL(string(src))
	if err != nil {
		return nil, err
	}
	var names []string
	for _, t := range tables {
		if t.Name == name {
			return t, nil
		}
		names = append(names, t.Name)
	}
	return nil, fmt.Errorf("table %q not found in %s (available: %s); use -table", name, file, strings.Join(names, ", "))
}

type fileSet struct{ request, response, service, controller, router, job, entity, dao string }

func files(n Name) fileSet {
	f := n.Snake + ".go"
	return fileSet{
		request:    "app/do/request/" + f,
		response:   "app/do/response/" + f,
		service:    "app/service/" + f,
		controller: "app/controller/" + f,
		router:     "app/router/" + f,
		job:        "app/daemon_jobs/cron/" + f,
		entity:     "app/modules/mysql/entity/" + f,
		dao:        "app/modules/mysql/dao/" + f,
	}
}

func normalizeComponents(in []string) ([]string, error) {
	if len(in) == 0 {
		return slices.Clone(DefaultComponents), nil
	}
	var out []string
	for _, c := range in {
		c = strings.ToLower(strings.TrimSpace(c))
		switch {
		case c == "":
		case c == "all":
			for _, a := range allComponents {
				if !slices.Contains(out, a) {
					out = append(out, a)
				}
			}
		case slices.Contains(allComponents, c):
			if !slices.Contains(out, c) {
				out = append(out, c)
			}
		default:
			return nil, fmt.Errorf("unknown component %q (valid: %s, all)", c, strings.Join(allComponents, ", "))
		}
	}
	return out, nil
}

var standardActions = map[string]struct{ http, path, summary string }{
	"list":   {"GET", "", "列表"},
	"get":    {"GET", "/:id", "详情"},
	"create": {"POST", "", "创建"},
	"update": {"PUT", "/:id", "更新"},
	"delete": {"DELETE", "/:id", "删除"},
}

func buildActions(n Name, names []string, summary string) ([]Action, error) {
	var out []Action
	seen := map[string]bool{}
	for _, raw := range names {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		an, err := NewName(raw)
		if err != nil {
			return nil, fmt.Errorf("action: %w", err)
		}
		if seen[an.Snake] {
			continue
		}
		seen[an.Snake] = true
		a := Action{Method: an.Pascal, Req: an.Pascal + n.Pascal}
		sep := ""
		if last := summary[len(summary)-1]; last < 0x80 {
			sep = " " // 英文名与中文动作之间加空格：Order 列表；中文表注释则直接拼接：订单列表
		}
		if std, ok := standardActions[an.Snake]; ok {
			a.Kind, a.HTTP, a.Path, a.Summary = an.Snake, std.http, std.path, summary+sep+std.summary
		} else {
			a.Kind, a.HTTP, a.Path, a.Summary = "custom", "POST", "/"+an.Kebab, summary+" "+an.Kebab
		}
		switch a.Kind {
		case "list":
			a.Resp = n.Pascal + "List"
		case "get", "create", "update":
			a.Resp = n.Pascal
		case "delete":
			a.Resp = "Empty"
		default:
			a.Resp = an.Pascal + n.Pascal
		}
		a.HasID = strings.Contains(a.Path, ":id")
		a.HasBody = a.HTTP == "POST" || a.HTTP == "PUT"
		a.SwagPath = "/v1/" + n.Kebab + strings.ReplaceAll(a.Path, ":id", "{id}")
		out = append(out, a)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no actions")
	}
	return out, nil
}

func jobSchedule(n Name, every, spec string) (string, error) {
	opts := "scheduler.WithLock(time.Minute), scheduler.WithTimeout(30*time.Second)"
	if spec != "" {
		if _, err := cronSpecCheck(spec); err != nil {
			return "", err
		}
		return fmt.Sprintf("s.Cron(%q, %q, j.Run, %s)", n.Snake, spec, opts), nil
	}
	if every == "" {
		every = "5m"
	}
	d, err := time.ParseDuration(every)
	if err != nil || d <= 0 {
		return "", fmt.Errorf("invalid -every %q: use a duration like 30s, 5m, 1h", every)
	}
	return fmt.Sprintf("s.Every(%q, %s, j.Run, %s)", n.Snake, durationExpr(d), opts), nil
}

func durationExpr(d time.Duration) string {
	for _, u := range []struct {
		d    time.Duration
		name string
	}{{time.Hour, "time.Hour"}, {time.Minute, "time.Minute"}, {time.Second, "time.Second"}, {time.Millisecond, "time.Millisecond"}} {
		if d%u.d == 0 {
			if d == u.d {
				return u.name
			}
			return fmt.Sprintf("%d*%s", d/u.d, u.name)
		}
	}
	return fmt.Sprintf("time.Duration(%d)", d)
}

// renderGo 渲染模板并整理 import、gofmt
func renderGo(rel, name string, v any) ([]byte, error) {
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, name, v); err != nil {
		return nil, fmt.Errorf("render %s: %w", rel, err)
	}
	out, err := imports.Process(rel, buf.Bytes(), &imports.Options{Comments: true, TabIndent: true, TabWidth: 8, FormatOnly: false})
	if err != nil {
		return nil, fmt.Errorf("format %s: %w\n%s", rel, err, buf.String())
	}
	return out, nil
}
