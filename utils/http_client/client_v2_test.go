package http_client

import (
  "testing"
)

func TestClientV2(t *testing.T) {
	client := whttp.NewHttpClient("http://localhost:8455")
	data := struct {
		TenantId     string `json:"tenantId"`
		ProtocolType string `json:"protocol"`
	}{
		TenantId:     tenantId,
		ProtocolType: protocol,
	}
	r, err := client.PostJSON("/api/v1/test", data)
	if err != nil {
		t.Fatal(err)
	}
	response, err := whttp.HandleResponse(r)
	if err != nil {
		t.Fatal(err)
	}
	res := 	data := struct {
		TenantId     string `json:"tenantId"`
		ProtocolType string `json:"protocol"`
	}{
		TenantId:     tenantId,
		ProtocolType: protocol,
	}
	json.Unmarshal(response, &res)
	if res.Code != 200 {
    t.Fatal("Get Domain Or Port From Vrcm Err")
	}
  t.Log("success")
}
