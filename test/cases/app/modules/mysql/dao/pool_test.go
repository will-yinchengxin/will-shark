package dao

import (
	"strings"
	"testing"
	"willshark/app/do/request"
	"willshark/consts"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestPaginateNumber(t *testing.T) {
	cases := []struct {
		in                     request.Page
		page, offset, pageSize int
	}{
		{request.Page{}, 1, 0, consts.PageSizeDefault},
		{request.Page{Page: 3, Size: 20}, 3, 40, 20},
		{request.Page{Page: -2, Size: -1}, 1, 0, consts.PageSizeDefault},
		{request.Page{Page: 2, Size: 1000000}, 2, consts.PageSizeMax, consts.PageSizeMax},
	}
	for _, c := range cases {
		p, o, s := PaginateNumber(c.in)
		if p != c.page || o != c.offset || s != c.pageSize {
			t.Errorf("PaginateNumber(%+v) = %d,%d,%d want %d,%d,%d", c.in, p, o, s, c.page, c.offset, c.pageSize)
		}
	}
}

func TestPaginateScopes(t *testing.T) {
	db, _ := gorm.Open(mysql.New(mysql.Config{DSN: "u@tcp(127.0.0.1:1)/d", SkipInitializeWithVersion: true}), &gorm.Config{DryRun: true, DisableAutomaticPing: true})
	var rows []map[string]any
	stmt := db.Table("t").Scopes(Paginate(request.Page{Page: 3, Size: 10})).Find(&rows).Statement
	if sql := stmt.SQL.String(); !strings.Contains(sql, "LIMIT ?") || !strings.Contains(sql, "OFFSET ?") || stmt.Vars[0] != 10 || stmt.Vars[1] != 20 {
		t.Fatalf("sql=%s vars=%v", sql, stmt.Vars)
	}
	stmt = db.Table("t").Scopes(PaginateParams(5, 7)).Find(&rows).Statement
	if stmt.Vars[0] != 7 || stmt.Vars[1] != 5 {
		t.Fatalf("vars=%v", stmt.Vars)
	}
}
