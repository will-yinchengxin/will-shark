package gen

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Table 从 CREATE TABLE 解析出的表结构
type Table struct {
	Name    string
	Comment string
	Columns []Column
	PK      []string
}

type Column struct {
	Name     string
	DataType string // 小写，如 varchar
	Size     int    // varchar(255) → 255；tinyint(1) → 1
	Unsigned bool
	NotNull  bool
	AutoInc  bool
	HasDef   bool
	Comment  string
}

var createRe = regexp.MustCompile("(?is)CREATE\\s+TABLE\\s+(?:IF\\s+NOT\\s+EXISTS\\s+)?((?:`[^`]+`|\\w+)(?:\\s*\\.\\s*(?:`[^`]+`|\\w+))?)\\s*\\(")

// ParseSQL 解析 SQL 文本中的所有 CREATE TABLE 语句（MySQL 方言）
func ParseSQL(src string) ([]*Table, error) {
	src = stripComments(src)
	var tables []*Table
	pos := 0
	for {
		loc := createRe.FindStringSubmatchIndex(src[pos:])
		if loc == nil {
			break
		}
		name := unquoteIdent(lastPart(src[pos+loc[2] : pos+loc[3]]))
		bodyStart := pos + loc[1] // '(' 之后
		bodyEnd, err := matchParen(src, bodyStart)
		if err != nil {
			return nil, fmt.Errorf("table %s: %w", name, err)
		}
		t := &Table{Name: name}
		for _, def := range splitTopLevel(src[bodyStart:bodyEnd]) {
			if err := t.addDefinition(def); err != nil {
				return nil, fmt.Errorf("table %s: %w", name, err)
			}
		}
		rest := src[bodyEnd+1:]
		if end := indexOutsideQuotes(rest, ';'); end >= 0 {
			rest = rest[:end]
		}
		t.Comment = optionComment(rest)
		if len(t.Columns) == 0 {
			return nil, fmt.Errorf("table %s: no columns", name)
		}
		tables = append(tables, t)
		pos = bodyEnd + 1
	}
	if len(tables) == 0 {
		return nil, fmt.Errorf("no CREATE TABLE statement found")
	}
	return tables, nil
}

func (t *Table) Column(name string) *Column {
	for i := range t.Columns {
		if t.Columns[i].Name == name {
			return &t.Columns[i]
		}
	}
	return nil
}

var skipPrefixes = []string{"KEY", "INDEX", "UNIQUE", "CONSTRAINT", "FULLTEXT", "SPATIAL", "FOREIGN", "CHECK"}

func (t *Table) addDefinition(def string) error {
	def = strings.TrimSpace(def)
	if def == "" {
		return nil
	}
	upper := strings.ToUpper(def)
	if strings.HasPrefix(upper, "PRIMARY KEY") {
		t.PK = parseIdentList(def)
		return nil
	}
	for _, p := range skipPrefixes {
		if strings.HasPrefix(upper, p+" ") || strings.HasPrefix(upper, p+"(") {
			return nil
		}
	}

	name, rest := takeIdent(def)
	if name == "" {
		return fmt.Errorf("cannot parse column definition %q", def)
	}
	c := Column{Name: name}
	rest = strings.TrimSpace(rest)
	i := 0
	for i < len(rest) && (isIdentChar(rest[i])) {
		i++
	}
	c.DataType = strings.ToLower(rest[:i])
	rest = rest[i:]
	if strings.HasPrefix(rest, "(") {
		end, err := matchParen(rest, 1)
		if err != nil {
			return fmt.Errorf("column %s: %w", name, err)
		}
		args := rest[1:end]
		if n, err := strconv.Atoi(strings.TrimSpace(strings.Split(args, ",")[0])); err == nil {
			c.Size = n
		}
		rest = rest[end+1:]
	}
	attrs := strings.ToUpper(stripQuoted(rest))
	c.Unsigned = strings.Contains(attrs, "UNSIGNED")
	c.NotNull = strings.Contains(attrs, "NOT NULL")
	c.AutoInc = strings.Contains(attrs, "AUTO_INCREMENT")
	c.HasDef = regexp.MustCompile(`\bDEFAULT\b`).MatchString(attrs) && !regexp.MustCompile(`\bDEFAULT\s+NULL\b`).MatchString(attrs)
	if strings.Contains(attrs, "PRIMARY KEY") {
		t.PK = []string{name}
		c.NotNull = true
	}
	c.Comment = columnComment(rest)
	t.Columns = append(t.Columns, c)
	return nil
}

var commentRe = regexp.MustCompile(`(?is)\bCOMMENT\s*=?\s*'((?:[^'\\]|\\.|'')*)'`)

// columnComment 提取 COMMENT '...'；关键字须在引号外，避免误匹配 DEFAULT 'xx comment'
func columnComment(s string) string {
	loc := commentKeywordRe.FindStringIndex(maskQuoted(s))
	if loc == nil {
		return ""
	}
	m := commentRe.FindStringSubmatch(s[loc[0]:])
	if m == nil {
		return ""
	}
	return cleanComment(m[1])
}

var commentKeywordRe = regexp.MustCompile(`(?i)\bCOMMENT\s*=?\s*'`)

// maskQuoted 把引号内的字符替换为空格（长度不变），便于在原串上按位置匹配引号外的关键字
func maskQuoted(s string) string {
	b := []byte(s)
	var quote byte
	for i := 0; i < len(b); i++ {
		ch := b[i]
		if quote != 0 {
			if ch == '\\' && i+1 < len(b) {
				b[i], b[i+1] = ' ', ' '
				i++
			} else if ch == quote {
				quote = 0
			} else {
				b[i] = ' '
			}
			continue
		}
		if ch == '\'' || ch == '"' {
			quote = ch
		}
	}
	return string(b)
}

func optionComment(s string) string { return columnComment(s) }

func cleanComment(s string) string {
	s = strings.ReplaceAll(s, "''", "'")
	s = strings.ReplaceAll(s, `\'`, "'")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(s)
}

// matchParen 从 start（'(' 之后的位置）开始找到匹配的 ')'，忽略引号内的括号
func matchParen(s string, start int) (int, error) {
	depth := 1
	var quote byte
	for i := start; i < len(s); i++ {
		ch := s[i]
		if quote != 0 {
			if ch == '\\' {
				i++
			} else if ch == quote {
				quote = 0
			}
			continue
		}
		switch ch {
		case '\'', '"', '`':
			quote = ch
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i, nil
			}
		}
	}
	return 0, fmt.Errorf("unbalanced parentheses")
}

// splitTopLevel 按顶层逗号拆分（忽略括号与引号内的逗号）
func splitTopLevel(s string) []string {
	var parts []string
	depth, last := 0, 0
	var quote byte
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if quote != 0 {
			if ch == '\\' {
				i++
			} else if ch == quote {
				quote = 0
			}
			continue
		}
		switch ch {
		case '\'', '"', '`':
			quote = ch
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, s[last:i])
				last = i + 1
			}
		}
	}
	return append(parts, s[last:])
}

func indexOutsideQuotes(s string, target byte) int {
	var quote byte
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if quote != 0 {
			if ch == '\\' {
				i++
			} else if ch == quote {
				quote = 0
			}
			continue
		}
		if ch == '\'' || ch == '"' || ch == '`' {
			quote = ch
		} else if ch == target {
			return i
		}
	}
	return -1
}

// stripComments 去掉引号外的 "-- "、"#" 行注释与 /* */ 块注释（换行保留）
func stripComments(s string) string {
	var b strings.Builder
	var quote byte
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if quote != 0 {
			b.WriteByte(ch)
			if ch == '\\' && i+1 < len(s) {
				i++
				b.WriteByte(s[i])
			} else if ch == quote {
				quote = 0
			}
			continue
		}
		switch {
		case ch == '\'' || ch == '"' || ch == '`':
			quote = ch
			b.WriteByte(ch)
		case ch == '#' || (ch == '-' && strings.HasPrefix(s[i:], "--") && (i+2 == len(s) || s[i+2] == ' ' || s[i+2] == '\t' || s[i+2] == '\n' || s[i+2] == '\r')):
			for i < len(s) && s[i] != '\n' {
				i++
			}
			if i < len(s) {
				b.WriteByte('\n')
			}
		case ch == '/' && strings.HasPrefix(s[i:], "/*"):
			end := strings.Index(s[i+2:], "*/")
			if end < 0 {
				return b.String()
			}
			i += end + 3
			b.WriteByte(' ')
		default:
			b.WriteByte(ch)
		}
	}
	return b.String()
}

// stripQuoted 去掉引号内的内容，避免注释中的关键字干扰属性识别
func stripQuoted(s string) string {
	var b strings.Builder
	var quote byte
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if quote != 0 {
			if ch == '\\' {
				i++
			} else if ch == quote {
				quote = 0
			}
			continue
		}
		if ch == '\'' || ch == '"' {
			quote = ch
			b.WriteString("''")
			continue
		}
		b.WriteByte(ch)
	}
	return b.String()
}

func takeIdent(s string) (string, string) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "`") {
		end := strings.Index(s[1:], "`")
		if end < 0 {
			return "", s
		}
		return s[1 : end+1], s[end+2:]
	}
	i := 0
	for i < len(s) && isIdentChar(s[i]) {
		i++
	}
	return s[:i], s[i:]
}

func parseIdentList(def string) []string {
	open := strings.Index(def, "(")
	if open < 0 {
		return nil
	}
	end, err := matchParen(def, open+1)
	if err != nil {
		return nil
	}
	var cols []string
	for _, p := range splitTopLevel(def[open+1 : end]) {
		name, _ := takeIdent(p)
		if name != "" {
			cols = append(cols, name)
		}
	}
	return cols
}

func isIdentChar(c byte) bool {
	return c == '_' || c == '$' || (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func unquoteIdent(s string) string { return strings.Trim(strings.TrimSpace(s), "`") }

var identRe = regexp.MustCompile("`[^`]+`|\\w+")

// lastPart 取 db.table / `db`.`table` 中的表名
func lastPart(s string) string {
	ids := identRe.FindAllString(s, -1)
	if len(ids) == 0 {
		return s
	}
	return ids[len(ids)-1]
}
