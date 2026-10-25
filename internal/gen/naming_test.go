package gen

import "testing"

func TestNewName(t *testing.T) {
	cases := []struct{ in, pascal, camel, snake, kebab string }{
		{"order", "Order", "order", "order", "order"},
		{"order_item", "OrderItem", "orderItem", "order_item", "order-item"},
		{"order-item", "OrderItem", "orderItem", "order_item", "order-item"},
		{"OrderItem", "OrderItem", "orderItem", "order_item", "order-item"},
		{"tenantId", "TenantID", "tenantID", "tenant_id", "tenant-id"},
		{"HTTPServer", "HTTPServer", "httpServer", "http_server", "http-server"},
		{"user_ip", "UserIP", "userIP", "user_ip", "user-ip"},
		{"ann_log_statistics", "AnnLogStatistics", "annLogStatistics", "ann_log_statistics", "ann-log-statistics"},
		{"type", "Type", "type_", "type", "type"},
		{"v2_api", "V2API", "v2API", "v2_api", "v2-api"},
	}
	for _, c := range cases {
		n, err := NewName(c.in)
		if err != nil {
			t.Fatalf("%s: %v", c.in, err)
		}
		if n.Pascal != c.pascal || n.Camel != c.camel || n.Snake != c.snake || n.Kebab != c.kebab {
			t.Errorf("%s: got %+v", c.in, n)
		}
	}
	for _, bad := range []string{"", "_", "1order", "--"} {
		if _, err := NewName(bad); err == nil {
			t.Errorf("%q should be invalid", bad)
		}
	}
}

func TestFieldName(t *testing.T) {
	for in, want := range map[string]string{"id": "ID", "job_id": "JobID", "cpeId": "CpeID", "TableName": "TableNameField", "2fa": "F2fa"} {
		if got := fieldName(in); got != want {
			t.Errorf("fieldName(%q) = %q, want %q", in, got, want)
		}
	}
}
