package gen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateIncludesNonAutoPrimaryKey(t *testing.T) {
	g, dir := newGen(t)
	sql := filepath.Join(dir, "keys.sql")
	if err := os.WriteFile(sql, []byte("CREATE TABLE ticket (code varchar(24) NOT NULL PRIMARY KEY, title varchar(50) NOT NULL);"), 0o600); err != nil {
		t.Fatal(err)
	}
	apply(t, g, ModuleOptions{Name: "ticket", SQLFile: sql}, false)
	got := read(t, filepath.Join(dir, "app/do/request/ticket.go"))
	start := strings.Index(got, "type CreateTicket struct")
	end := strings.Index(got[start:], "\n}")
	if !strings.Contains(got[start:start+end], `json:"code"`) {
		t.Fatalf("non-auto PK missing from create request:\n%s", got)
	}
	service := read(t, filepath.Join(dir, "app/service/ticket.go"))
	if !strings.Contains(strings.Join(strings.Fields(service), " "), "Code: req.Code") {
		t.Fatalf("create does not assign PK:\n%s", service)
	}
}

func TestDuplicateGoFieldsRejected(t *testing.T) {
	tables, err := ParseSQL("CREATE TABLE collision (id bigint PRIMARY KEY, user_id int, userID int);")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := newTableData(tables[0]); err == nil {
		t.Fatal("colliding Go fields must fail before generation")
	}
}

func TestModelCanKeepNonPrimaryIDColumn(t *testing.T) {
	g, dir := newGen(t)
	file := filepath.Join(dir, "model.sql")
	if err := os.WriteFile(file, []byte("CREATE TABLE indexed (code varchar(8) PRIMARY KEY, id int);"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := g.Model(ModelOptions{SQLFile: file}); err != nil {
		t.Fatalf("model has no path DTO and must allow this schema: %v", err)
	}
}
