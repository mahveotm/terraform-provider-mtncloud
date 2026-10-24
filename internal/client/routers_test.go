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
			fmt.Fprint(w, `{"networkRouterTypes":[{"id":2,"code":"neutron","name":"Neutron"}]}`)
		case "GET /api/network-router-types/2":
			fmt.Fprint(w, `{"networkRouterType":{"id":2,"code":"neutron","creatable":true,"optionTypes":[{"fieldName":"enableSnat"}]}}`)
		case "POST /api/networks/routers":
			var body struct {
				Router map[string]json.RawMessage `json:"networkRouter"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if string(body.Router["enabled"]) != "false" || string(body.Router["site"]) != `{"id":4}` || string(body.Router["config"]) != `{"enableSnat":false}` {
				t.Errorf("incorrect payload: %s", body.Router)
			}
			fmt.Fprint(w, `{"success":true,"id":42}`)
		case "GET /api/networks/routers/42":
			fmt.Fprint(w, `{"networkRouter":{"id":42,"name":"app","enabled":false,"type":{"id":2,"code":"neutron"},"site":{"id":4},"zone":{"id":8}}}`)
		case "PUT /api/networks/routers/42":
			var body struct {
				Router map[string]any `json:"networkRouter"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if len(body.Router) != 2 || body.Router["name"] != "renamed" || body.Router["enabled"] != false {
				t.Errorf("incorrect update: %#v", body.Router)
			}
			fmt.Fprint(w, `{"success":true}`)
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
	typ, err := c.GetRouterTypeByName(ctx, "neutron")
	if err != nil || !typ.Creatable || len(typ.OptionTypes) != 1 {
		t.Fatalf("type: %#v, %v", typ, err)
	}
	fields, err := RouterOptions(`{"config":{"enableSnat":false}}`)
	if err != nil {
		t.Fatal(err)
	}
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
	if err := c.UpdateRouter(ctx, id, "renamed", false); err != nil {
		t.Fatal(err)
	}
	if err := c.DeleteRouter(ctx, id); err != nil {
		t.Fatal(err)
	}
}

func TestRouterOptionsRejectInvalidAndCoreFields(t *testing.T) {
	for _, raw := range []string{`null`, `[]`, `{"name":"override"}`, `{"config":{}} {}`} {
		if _, err := RouterOptions(raw); err == nil {
			t.Errorf("accepted %s", raw)
		}
	}
}
