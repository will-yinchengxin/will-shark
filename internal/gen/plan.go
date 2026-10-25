package gen

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

// Plan 一次生成的全部变更；Apply 前不会修改任何文件
type Plan struct {
	gen     *Generator
	creates []fileCreate
	edits   []*fileEdit
	notes   []string
}

type fileCreate struct {
	rel     string
	content []byte
}

type fileEdit struct {
	rel     string
	inserts []insertion
	imports []string
}

type insertion struct{ marker, line string }

func (p *Plan) create(rel string, content []byte) {
	p.creates = append(p.creates, fileCreate{rel: rel, content: content})
}

func (p *Plan) edit(rel string) *fileEdit {
	for _, e := range p.edits {
		if e.rel == rel {
			return e
		}
	}
	e := &fileEdit{rel: rel}
	p.edits = append(p.edits, e)
	return e
}

func (p *Plan) insert(rel, marker, line string) {
	e := p.edit(rel)
	e.inserts = append(e.inserts, insertion{marker: marker, line: line})
}

func (p *Plan) addImport(rel, path string) {
	e := p.edit(rel)
	if !slices.Contains(e.imports, path) {
		e.imports = append(e.imports, path)
	}
}

func (p *Plan) note(format string, a ...any) { p.notes = append(p.notes, fmt.Sprintf(format, a...)) }

// Files 将要新建（或覆盖）的文件，相对项目根目录
func (p *Plan) Files() []string {
	var out []string
	for _, c := range p.creates {
		out = append(out, c.rel)
	}
	return out
}

// Apply 写入变更。force=false 时任何目标文件已存在都会中止且不写入任何内容；
// dryRun=true 只打印计划。插入点代码已存在时跳过（可重复执行）。
func (p *Plan) Apply(w io.Writer, force, dryRun bool) error {
	var conflicts []string
	for _, c := range p.creates {
		if p.gen.exists(c.rel) && !force {
			conflicts = append(conflicts, c.rel)
		}
	}
	if len(conflicts) > 0 {
		return fmt.Errorf("files already exist (use -force to overwrite):\n  %s", strings.Join(conflicts, "\n  "))
	}
	if err := p.checkRedeclared(); err != nil {
		return err
	}

	// 先在内存中完成所有编辑，任何一步失败都不落盘
	edited := map[string][]byte{}
	var editSummary []string
	for _, e := range p.edits {
		out, added, err := p.applyEdit(e)
		if err != nil {
			return err
		}
		if added > 0 {
			edited[e.rel] = out
			editSummary = append(editSummary, fmt.Sprintf("~ update  %s (+%d)", e.rel, added))
		} else {
			editSummary = append(editSummary, fmt.Sprintf("= skip    %s (already wired)", e.rel))
		}
	}

	prefix := ""
	if dryRun {
		prefix = "[dry-run] "
	}
	for _, c := range p.creates {
		verb := "+ create "
		if p.gen.exists(c.rel) {
			verb = "! replace"
		}
		fmt.Fprintf(w, "%s%s %s\n", prefix, verb, c.rel)
	}
	for _, s := range editSummary {
		fmt.Fprintf(w, "%s%s\n", prefix, s)
	}
	for _, n := range p.notes {
		fmt.Fprintf(w, "%s* %s\n", prefix, n)
	}
	if dryRun {
		return nil
	}

	for _, c := range p.creates {
		abs := p.gen.path(c.rel)
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil { // #nosec G301 -- 生成的源码目录需对团队可读，与 git 检出的目录权限一致
			return err
		}
		if err := os.WriteFile(abs, c.content, 0o644); err != nil { // #nosec G306 -- 生成的是源码文件，权限与仓库中其他源码一致（0644）
			return err
		}
	}
	for rel, out := range edited {
		if err := os.WriteFile(p.gen.path(rel), out, 0o644); err != nil { // #nosec G306 -- 修改的是已存在的源码文件，权限与仓库中其他源码一致（0644）
			return err
		}
	}
	return nil
}

// applyEdit 在 "// <marker>" 所在行之前插入代码（沿用该行缩进），已存在的行跳过
func (p *Plan) applyEdit(e *fileEdit) ([]byte, int, error) {
	src, err := os.ReadFile(p.gen.path(e.rel))
	if err != nil {
		return nil, 0, err
	}
	lines := strings.Split(string(src), "\n")
	existing := map[string]bool{}
	for _, l := range lines {
		existing[strings.TrimSpace(l)] = true
	}
	added := 0
	for _, ins := range e.inserts {
		if existing[strings.TrimSpace(ins.line)] {
			continue
		}
		idx := slices.IndexFunc(lines, func(l string) bool { return strings.TrimSpace(l) == "// "+ins.marker })
		if idx < 0 {
			return nil, 0, fmt.Errorf("%s: marker \"// %s\" not found; add it back or use -no-wire and wire manually", e.rel, ins.marker)
		}
		indent := lines[idx][:len(lines[idx])-len(strings.TrimLeft(lines[idx], " \t"))]
		lines = slices.Insert(lines, idx, indent+ins.line)
		existing[strings.TrimSpace(ins.line)] = true
		added++
	}
	out := []byte(strings.Join(lines, "\n"))

	if len(e.imports) > 0 {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, e.rel, out, parser.ParseComments)
		if err != nil {
			return nil, 0, fmt.Errorf("%s: %w", e.rel, err)
		}
		for _, imp := range e.imports {
			if astutil.AddImport(fset, f, imp) {
				added++
			}
		}
		var buf bytes.Buffer
		if err := format.Node(&buf, fset, f); err != nil {
			return nil, 0, err
		}
		out = buf.Bytes()
	}
	if added == 0 {
		return src, 0, nil
	}
	formatted, err := format.Source(out)
	if err != nil {
		return nil, 0, fmt.Errorf("%s: generated code does not parse: %w", e.rel, err)
	}
	return formatted, added, nil
}

// checkRedeclared 新文件中的包级标识符不能与同一目录（同一包）中其他文件重复，否则生成后无法编译，
// 例如 module empty 生成的 response.Empty 与 app/do/response/common.go 冲突。
func (p *Plan) checkRedeclared() error {
	byDir := map[string]map[string][]byte{} // dir → rel → content（新建的文件）
	for _, c := range p.creates {
		if strings.HasSuffix(c.rel, ".go") {
			dir := filepath.Dir(c.rel)
			if byDir[dir] == nil {
				byDir[dir] = map[string][]byte{}
			}
			byDir[dir][c.rel] = c.content
		}
	}
	var problems []string
	for dir, created := range byDir {
		owner := map[string]string{} // 标识符 → 声明它的文件
		record := func(rel string, src []byte, isNew bool) error {
			names, err := topLevelNames(rel, src)
			if err != nil {
				return err
			}
			for _, n := range names {
				if prev, dup := owner[n]; dup && (isNew || created[prev] != nil) {
					problems = append(problems, fmt.Sprintf("%s: %s already declared in %s", rel, n, prev))
					continue
				}
				owner[n] = rel
			}
			return nil
		}
		entries, _ := os.ReadDir(p.gen.path(dir))
		for _, e := range entries {
			rel := filepath.ToSlash(filepath.Join(dir, e.Name()))
			if e.IsDir() || !strings.HasSuffix(rel, ".go") || created[rel] != nil {
				continue // 将被覆盖的文件以新内容为准
			}
			src, err := os.ReadFile(p.gen.path(rel))
			if err != nil {
				return err
			}
			if err := record(rel, src, false); err != nil {
				return err
			}
		}
		rels := make([]string, 0, len(created))
		for rel := range created {
			rels = append(rels, rel)
		}
		slices.Sort(rels)
		for _, rel := range rels {
			if err := record(rel, created[rel], true); err != nil {
				return err
			}
		}
	}
	if len(problems) > 0 {
		slices.Sort(problems)
		return fmt.Errorf("generated code would not compile (choose another module name):\n  %s", strings.Join(problems, "\n  "))
	}
	return nil
}

// topLevelNames 文件中的包级类型、变量、常量、函数名（不含方法；_test 外部测试包返回空）
func topLevelNames(rel string, src []byte) ([]string, error) {
	f, err := parser.ParseFile(token.NewFileSet(), rel, src, parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", rel, err)
	}
	if strings.HasSuffix(f.Name.Name, "_test") {
		return nil, nil
	}
	var names []string
	add := func(n string) {
		if n != "_" && n != "init" {
			names = append(names, n)
		}
	}
	for _, d := range f.Decls {
		switch d := d.(type) {
		case *ast.FuncDecl:
			if d.Recv == nil {
				add(d.Name.Name)
			}
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch sp := spec.(type) {
				case *ast.TypeSpec:
					add(sp.Name.Name)
				case *ast.ValueSpec:
					for _, n := range sp.Names {
						add(n.Name)
					}
				}
			}
		}
	}
	return names, nil
}
