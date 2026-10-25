package gen

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// Field 一个列在生成代码中的表示
type Field struct {
	Name     string // Go 字段名
	Column   string
	Type     string // 实体字段类型（可空列为指针）
	BaseType string // 去掉指针后的类型
	JSON     string
	Label    string
	Comment  string
	GormTag  string
	PK       bool
	Auto     bool // 自增或自动维护的时间列，不允许客户端写入

	CreateBinding string
	UpdateBinding string
}

// TableData 生成 CRUD 所需的表信息
type TableData struct {
	Name    string // 表名
	Comment string
	PK      Field
	Fields  []Field
	Create  []Field // 创建接口可写字段
	Update  []Field // 更新接口可写字段
}

var (
	createTimeCols = map[string]bool{"create_time": true, "created_at": true, "create_at": true, "gmt_create": true, "ctime": true, "createtime": true, "createdat": true}
	updateTimeCols = map[string]bool{"update_time": true, "updated_at": true, "update_at": true, "gmt_modified": true, "mtime": true, "updatetime": true, "updatedat": true}
)

func newTableData(t *Table) (*TableData, error) {
	td := &TableData{Name: t.Name, Comment: t.Comment}
	pkSet := map[string]bool{}
	for _, c := range t.PK {
		pkSet[c] = true
	}
	for _, c := range t.Columns {
		f := newField(c, pkSet[c.Name])
		td.Fields = append(td.Fields, f)
		if f.PK {
			td.PK = f
		}
		if f.PK || f.Auto {
			continue
		}
		td.Create = append(td.Create, f)
		td.Update = append(td.Update, f)
	}
	if len(t.PK) != 1 || td.PK.Name == "" {
		return td, fmt.Errorf("table %s must have a single-column primary key to generate CRUD (use the model command for entity/dao only)", t.Name)
	}
	return td, nil
}

func newField(c Column, pk bool) Field {
	base, nullable := goType(c)
	f := Field{
		Name:     fieldName(c.Name),
		Column:   c.Name,
		BaseType: base,
		Type:     base,
		JSON:     c.Name,
		Comment:  c.Comment,
		PK:       pk,
	}
	if nullable && !pk && base != "[]byte" {
		f.Type = "*" + base
	}
	f.Label = labelOf(c)

	tags := []string{"column:" + c.Name}
	if pk {
		tags = append(tags, "primaryKey")
	}
	lower := strings.ToLower(c.Name)
	switch {
	case c.AutoInc:
		tags = append(tags, "autoIncrement")
		f.Auto = true
	case createTimeCols[lower] && base == "time.Time":
		tags = append(tags, "autoCreateTime")
		f.Auto = true
	case updateTimeCols[lower] && base == "time.Time":
		tags = append(tags, "autoUpdateTime")
		f.Auto = true
	}
	f.GormTag = strings.Join(tags, ";")

	var rules []string
	if c.NotNull && !c.HasDef && !c.AutoInc && base != "bool" {
		rules = append(rules, "required")
	} else {
		rules = append(rules, "omitempty")
	}
	if base == "string" && c.Size > 0 && isCharType(c.DataType) {
		rules = append(rules, "max="+strconv.Itoa(c.Size))
	}
	f.CreateBinding = strings.Join(rules, ",")
	if f.CreateBinding == "omitempty" {
		f.CreateBinding = ""
	}
	f.UpdateBinding = strings.Join(append([]string{"omitempty"}, rules[1:]...), ",")
	if f.UpdateBinding == "omitempty" {
		f.UpdateBinding = ""
	}
	return f
}

func labelOf(c Column) string {
	if c.Comment != "" {
		// 取注释第一段作为字段中文名，如 "状态 1启用 2禁用" → "状态"
		label := strings.FieldsFunc(c.Comment, func(r rune) bool {
			return r == ' ' || r == ',' || r == '，' || r == ':' || r == '：' || r == '(' || r == '（' || r == ';' || r == '；'
		})
		// 跳过 "1 执行多次" 这类以数字开头的片段
		if len(label) > 0 && len([]rune(label[0])) <= 12 && !unicode.IsDigit([]rune(label[0])[0]) {
			return strings.ReplaceAll(label[0], `"`, "")
		}
	}
	return c.Name
}

func isCharType(t string) bool { return t == "varchar" || t == "char" }

// goType MySQL 类型 → Go 类型；第二个返回值表示是否可空
func goType(c Column) (string, bool) {
	u := ""
	if c.Unsigned {
		u = "u"
	}
	var t string
	switch c.DataType {
	case "tinyint":
		t = u + "int8"
	case "smallint", "year":
		t = u + "int16"
	case "mediumint", "int", "integer":
		t = u + "int32"
	case "bigint":
		t = u + "int64"
	case "bit":
		if c.Size <= 1 {
			t = "bool"
		} else {
			t = "uint64"
		}
	case "bool", "boolean":
		t = "bool"
	case "float":
		t = "float32"
	case "double", "real":
		t = "float64"
	case "decimal", "numeric":
		t = "string" // 避免浮点精度问题；需要计算时可换成 shopspring/decimal
	case "date", "datetime", "timestamp":
		t = "time.Time"
	case "binary", "varbinary", "tinyblob", "blob", "mediumblob", "longblob":
		t = "[]byte"
	default: // char varchar text json enum set time ...
		t = "string"
	}
	return t, !c.NotNull
}
