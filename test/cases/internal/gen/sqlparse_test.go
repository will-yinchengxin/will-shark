package gen

import (
	"os"
	"testing"
	"time"
)

const tricky = "CREATE TABLE IF NOT EXISTS `shop`.`t_order` (\n" +
	"  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',\n" +
	"  `user_id` int NOT NULL COMMENT '用户ID, (外键), 非空',\n" +
	"  `amount` decimal(10,2) NOT NULL DEFAULT '0.00' COMMENT '金额',\n" +
	"  `status` tinyint(1) NOT NULL DEFAULT 1 COMMENT 'it''s status: 1启用,2禁用',\n" +
	"  `remark` varchar(500) DEFAULT NULL,\n" +
	"  `paid` bit(1) NOT NULL DEFAULT b'0',\n" +
	"  `raw` blob,\n" +
	"  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,\n" +
	"  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,\n" +
	"  PRIMARY KEY (`id`),\n" +
	"  UNIQUE KEY `uk_user` (`user_id`, `status`),\n" +
	"  KEY `idx_created` (`created_at`),\n" +
	"  CONSTRAINT `fk_user` FOREIGN KEY (`user_id`) REFERENCES `user` (`id`)\n" +
	") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='订单表';\n" +
	"create table plain (id int primary key, name varchar(20) not null);"

func TestParseTricky(t *testing.T) {
	tables, err := ParseSQL(tricky)
	if err != nil {
		t.Fatal(err)
	}
	if len(tables) != 2 {
		t.Fatalf("tables = %d", len(tables))
	}
	o := tables[0]
	if o.Name != "t_order" || o.Comment != "订单表" || len(o.Columns) != 9 || len(o.PK) != 1 || o.PK[0] != "id" {
		t.Fatalf("t_order parsed wrong: name=%s comment=%q cols=%d pk=%v", o.Name, o.Comment, len(o.Columns), o.PK)
	}
	id := o.Column("id")
	if !id.Unsigned || !id.AutoInc || !id.NotNull || id.DataType != "bigint" || id.Comment != "主键" {
		t.Fatalf("id: %+v", id)
	}
	if c := o.Column("user_id"); c.Comment != "用户ID, (外键), 非空" || c.HasDef {
		t.Fatalf("user_id: %+v", c)
	}
	if c := o.Column("status"); c.Size != 1 || !c.HasDef || c.Comment != "it's status: 1启用,2禁用" {
		t.Fatalf("status: %+v", c)
	}
	if c := o.Column("remark"); c.NotNull || c.HasDef || c.Size != 500 {
		t.Fatalf("remark: %+v", c)
	}

	p := tables[1]
	if p.Name != "plain" || len(p.PK) != 1 || p.PK[0] != "id" || !p.Column("name").NotNull {
		t.Fatalf("plain: %+v", p)
	}
}

func TestTableDataMapping(t *testing.T) {
	tables, _ := ParseSQL(tricky)
	td, err := newTableData(tables[0])
	if err != nil {
		t.Fatal(err)
	}
	types := map[string]string{}
	fields := map[string]Field{}
	for _, f := range td.Fields {
		types[f.Column] = f.Type
		fields[f.Column] = f
	}
	want := map[string]string{
		"id": "uint64", "user_id": "int32", "amount": "string", "status": "int8", "remark": "*string",
		"paid": "bool", "raw": "[]byte", "created_at": "time.Time", "updated_at": "time.Time",
	}
	for col, typ := range want {
		if types[col] != typ {
			t.Errorf("%s: type %s, want %s", col, types[col], typ)
		}
	}
	if td.PK.Name != "ID" || td.PK.GormTag != "column:id;primaryKey;autoIncrement" {
		t.Fatalf("pk: %+v", td.PK)
	}
	if fields["created_at"].GormTag != "column:created_at;autoCreateTime" || fields["updated_at"].GormTag != "column:updated_at;autoUpdateTime" {
		t.Fatal("auto time tags missing")
	}
	for _, f := range td.Create {
		if f.Column == "id" || f.Column == "created_at" || f.Column == "updated_at" {
			t.Fatalf("auto column %s must not be writable", f.Column)
		}
	}
	if f := fields["user_id"]; f.CreateBinding != "required" || f.CreateType != "*int32" || !f.CreateDeref {
		t.Errorf("user_id: required numeric must be a pointer so 0 is accepted: %+v", f)
	}
	if f := fields["remark"]; f.CreateType != "*string" || f.CreateDeref {
		t.Errorf("remark: %+v", f)
	}
	if b := fields["remark"].CreateBinding; b != "omitempty,max=500" {
		t.Errorf("remark binding = %q", b)
	}
	if fields["user_id"].Label != "用户ID" {
		t.Errorf("label = %q", fields["user_id"].Label)
	}
}

func TestParseIgnoresComments(t *testing.T) {
	src := "-- 订单表, user's orders (draft\n" +
		"/* 块注释 ( ' */\n" +
		"CREATE TABLE `t` (\n" +
		"  `id` bigint NOT NULL AUTO_INCREMENT, -- 主键, it's (pk\n" +
		"  # 名称 ' (\n" +
		"  `name` varchar(20) NOT NULL DEFAULT '--not # a comment' COMMENT 'a -- b',\n" +
		"  `n` int NOT NULL,\n" +
		"  PRIMARY KEY (`id`)\n" +
		") COMMENT='/* 表 */';"
	tables, err := ParseSQL(src)
	if err != nil {
		t.Fatal(err)
	}
	tb := tables[0]
	if len(tb.Columns) != 3 || tb.Comment != "/* 表 */" || tb.Columns[1].Comment != "a -- b" || !tb.Columns[1].HasDef {
		t.Fatalf("%+v", tb)
	}
}

func TestLabelSkipsNumbers(t *testing.T) {
	if got := labelOf(Column{Name: "do_once", Comment: "1 执行多次 2 执行一次"}); got != "do_once" {
		t.Fatalf("label = %q", got)
	}
}

func TestNoPrimaryKey(t *testing.T) {
	tables, err := ParseSQL("CREATE TABLE log (msg text);")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := newTableData(tables[0]); err == nil {
		t.Fatal("expected error for table without primary key")
	}
}

func TestParseRepoSQL(t *testing.T) {
	src, err := os.ReadFile("../../sql/dynamic.sql")
	if err != nil {
		t.Skip(err)
	}
	tables, err := ParseSQL(string(src))
	if err != nil {
		t.Fatal(err)
	}
	if len(tables) != 4 || tables[0].Name != "task" || len(tables[0].Columns) != 17 {
		t.Fatalf("got %d tables, first=%s cols=%d", len(tables), tables[0].Name, len(tables[0].Columns))
	}
}

func TestParseErrors(t *testing.T) {
	for _, src := range []string{"", "SELECT 1;", "CREATE TABLE x (id int"} {
		if _, err := ParseSQL(src); err == nil {
			t.Errorf("%q should fail", src)
		}
	}
}

func TestGoTypeAndDurationExpr(t *testing.T) {
	src := "CREATE TABLE db.`x` (`id` int unsigned primary key, `b` bit(8), `f` float, `d` double, `y` year, `m` mediumint, " +
		"`j` json, `n` numeric(5,2), `bl` boolean NOT NULL, `vb` varbinary(16))"
	tables, err := ParseSQL(src)
	if err != nil || tables[0].Name != "x" {
		t.Fatalf("%v %+v", err, tables)
	}
	want := map[string]string{"id": "uint32", "b": "*uint64", "f": "*float32", "d": "*float64", "y": "*int16",
		"m": "*int32", "j": "*string", "n": "*string", "bl": "bool", "vb": "[]byte"}
	td, err := newTableData(tables[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range td.Fields {
		if want[f.Column] != f.Type {
			t.Errorf("%s: %s, want %s", f.Column, f.Type, want[f.Column])
		}
	}
	for d, want := range map[string]string{"1h": "time.Hour", "90s": "90*time.Second", "1500ms": "1500*time.Millisecond", "1us": "time.Duration(1000)"} {
		pd, _ := time.ParseDuration(d)
		if got := durationExpr(pd); got != want {
			t.Errorf("durationExpr(%s) = %s, want %s", d, got, want)
		}
	}
	if tables[0].Column("nope") != nil || tables[0].Column("id") == nil {
		t.Fatal("Column lookup")
	}
}
