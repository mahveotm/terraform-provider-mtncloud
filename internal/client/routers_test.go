package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouterLifecycle(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method + " " + r.URL.Path {
		case "GET /api/network-router-types":
			fmt.Fprint(w, `{"networkRouterTypes":[{"id":2,"code":"neutronRouter","name":"Neutron Router"}]}`)
		case "GET /api/network-router-types/2":
			fmt.Fprint(w, `{"networkRouterType":{"id":2,"code":"neutronRouter","creatable":true,"optionTypes":[{"fieldName":"enableSnat"}]}}`)
		case "POST /api/networks/routers":
			var body struct {
				Router map[string]json.RawMessage `json:"networkRouter"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if string(body.Router["enabled"]) != "false" || string(body.Router["site"]) != `{"id":4}` || string(body.Router["poolId"]) != `"pool-208"` {
				t.Errorf("incorrect payload: %s", body.Router)
			}
			fmt.Fprint(w, `{"success":true,"id":42}`)
		case "GET /api/networks/routers/42":
			fmt.Fprint(w, `{"networkRouter":{"id":42,"name":"app","status":"failed","enabled":false,"type":{"id":2,"code":"neutronRouter"},"site":{"id":4},"zone":{"id":8}}}`)
		case "DELETE /api/networks/routers/42":
			fmt.Fprint(w, `{"success":true}`)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	c, err := New(Config{URL: server.URL, Token: "test"})
	if err != nil {
		t.Fatal(err)
	}
	typ, err := c.GetRouterTypeByName(ctx, "neutronRouter")
	if err != nil || !typ.Creatable || len(typ.OptionTypes) != 1 {
		t.Fatalf("type: %#v, %v", typ, err)
	}
	fields := map[string]any{"poolId": "pool-208", "interfaces": []map[string]any{{"networkSubnet": map[string]any{"id": 4454}}}}
	fields["name"] = "app"
	fields["enabled"] = false
	fields["site"] = map[string]any{"id": 4}
	fields["type"] = map[string]any{"id": 2}
	fields["zone"] = map[string]any{"id": 8}
	id, err := c.CreateRouter(ctx, fields)
	if err != nil || id != 42 {
		t.Fatalf("create: %d %v", id, err)
	}
	router, err := c.GetRouter(ctx, id)
	if err != nil || router.Name != "app" {
		t.Fatalf("read: %#v %v", router, err)
	}
	if _, err := c.WaitForRouter(ctx, id, false); err == nil {
		t.Fatal("failed backend provisioning was reported as ready")
	}
	if err := c.DeleteRouter(ctx, id); err != nil {
		t.Fatal(err)
	}
}
