package gen

import (
	"bytes"
	"fmt"
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
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(abs, c.content, 0o644); err != nil {
			return err
		}
	}
	for rel, out := range edited {
		if err := os.WriteFile(p.gen.path(rel), out, 0o644); err != nil {
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
