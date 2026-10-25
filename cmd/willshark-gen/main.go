// willshark-gen 代码生成器。
//
//	go tool willshark-gen module order                                   # router + controller + service + dto（CRUD 骨架）
//	go tool willshark-gen module order -with all                         # 再加定时任务
//	go tool willshark-gen module order -with service,job -every 10m      # 任意组合
//	go tool willshark-gen module report -actions export,summary          # 自定义接口
//	go tool willshark-gen module task -sql sql/tables.sql                # 从建表语句生成完整 CRUD
//	go tool willshark-gen model -sql sql/tables.sql -tables user,order   # 只生成 entity + dao
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"willshark/internal/gen"
)

const usage = `willshark-gen: 生成 router / controller / service / dto / 定时任务，或从建表 SQL 生成 entity / dao / CRUD

用法:
  willshark-gen module <name> [flags]   生成业务模块，并自动接入 bootstrap 与路由
  willshark-gen model  -sql <file> [flags]   只生成 entity 与 dao

组件（-with，可任意组合，逗号分隔）:
  router      路由，挂载到 /v1/<name>，绑定 controller 方法
  controller  管理 HTTP 参数绑定、校验、错误映射与响应
  service     业务逻辑
  dto         请求/响应结构体（app/do/request, app/do/response）
  job         定时任务（app/daemon_jobs/cron，自动注入同名 service）
  all         以上全部
  默认生成 router,controller,service,dto。
  依赖会自动补齐：router → controller → service → dto；已存在且已接入的组件直接复用。

示例:
  willshark-gen module order
  willshark-gen module order -with all -every 10m
  willshark-gen module order -with job -cron "0 3 * * *"
  willshark-gen module report -actions export,summary
  willshark-gen module task -sql sql/dynamic.sql
  willshark-gen module order -sql sql/tables.sql -table t_order -with all
  willshark-gen model -sql sql/dynamic.sql -tables task,ann_order

运行 "willshark-gen module -h" 查看全部参数。
`

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		if !errors.Is(err, flag.ErrHelp) {
			fmt.Fprintln(os.Stderr, "error:", err)
		}
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return flag.ErrHelp
	}
	switch args[0] {
	case "module":
		return runModule(args[1:], stdout, stderr)
	case "model":
		return runModel(args[1:], stdout, stderr)
	case "-h", "--help", "help":
		fmt.Fprint(stdout, usage)
		return nil
	default:
		fmt.Fprint(stderr, usage)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

type common struct {
	dir    string
	force  bool
	dryRun bool
}

func (c *common) register(fs *flag.FlagSet) {
	fs.StringVar(&c.dir, "dir", ".", "项目目录（向上查找 go.mod）")
	fs.BoolVar(&c.force, "force", false, "覆盖已存在的文件")
	fs.BoolVar(&c.dryRun, "dry-run", false, "只打印将要进行的变更，不写入")
}

func runModule(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("module", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		c       common
		with    = fs.String("with", strings.Join(gen.DefaultComponents, ","), "组件: router,controller,service,dto,job 或 all")
		actions = fs.String("actions", strings.Join(gen.DefaultActions, ","), "接口: list,get,create,update,delete 或自定义名称（生成 POST /<name>）")
		sqlFile = fs.String("sql", "", "建表 SQL 文件；指定后生成 entity/dao 与完整 CRUD 实现")
		table   = fs.String("table", "", "SQL 文件中的表名（默认与模块名相同）")
		every   = fs.String("every", "", "定时任务间隔，如 30s、5m、1h（默认 5m）")
		cronStr = fs.String("cron", "", `定时任务 cron 表达式，如 "0 3 * * *"（支持秒字段），优先于 -every`)
		noWire  = fs.Bool("no-wire", false, "不修改 bootstrap/app.go 与 app/router/base.go")
	)
	c.register(fs)
	fs.Usage = func() {
		fmt.Fprintln(stderr, "用法: willshark-gen module <name> [flags]")
		fs.PrintDefaults()
	}
	name, rest := splitName(args)
	if err := fs.Parse(rest); err != nil {
		return err
	}
	if name == "" && fs.NArg() > 0 {
		name = fs.Arg(0)
	}
	if name == "" {
		fs.Usage()
		return errors.New("module name is required")
	}

	g, err := gen.New(c.dir)
	if err != nil {
		return err
	}
	plan, err := g.Module(gen.ModuleOptions{
		Name:     name,
		With:     strings.Split(*with, ","),
		Actions:  strings.Split(*actions, ","),
		SQLFile:  *sqlFile,
		Table:    *table,
		JobEvery: *every,
		JobCron:  *cronStr,
		NoWire:   *noWire,
	})
	if err != nil {
		return err
	}
	if err := plan.Apply(stdout, c.force, c.dryRun); err != nil {
		return err
	}
	if !c.dryRun {
		fmt.Fprintln(stdout, "\n完成。下一步: make check（如需更新 swagger 文档见 README）")
	}
	return nil
}

func runModel(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("model", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		c       common
		sqlFile = fs.String("sql", "", "建表 SQL 文件（必填）")
		tables  = fs.String("tables", "", "只生成指定表，逗号分隔（默认全部）")
	)
	c.register(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *sqlFile == "" {
		fs.Usage()
		return errors.New("-sql is required")
	}
	g, err := gen.New(c.dir)
	if err != nil {
		return err
	}
	var list []string
	for _, t := range strings.Split(*tables, ",") {
		if t = strings.TrimSpace(t); t != "" {
			list = append(list, t)
		}
	}
	plan, err := g.Model(gen.ModelOptions{SQLFile: *sqlFile, Tables: list})
	if err != nil {
		return err
	}
	return plan.Apply(stdout, c.force, c.dryRun)
}

// splitName 支持 "module order -with x" 与 "module -with x order" 两种写法
func splitName(args []string) (string, []string) {
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		return args[0], args[1:]
	}
	return "", args
}
