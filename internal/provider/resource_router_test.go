package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Exercise Terraform's plan/state contract against the documented API shapes,
// including import and an in-place update. No live MTN infrastructure is used.
func TestRouterLifecycleWithMockAPI(t *testing.T) {
	t.Setenv("MTN_CLOUD_USERNAME", "")
	t.Setenv("MTN_CLOUD_PASSWORD", "")
	var mu sync.Mutex
	name := "app"
	enabled := true
	deleted := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch r.Method + " " + r.URL.Path {
		case "GET /api/groups", "GET /api/groups/4":
			fmt.Fprint(w, `{"groups":[{"id":4,"name":"group","zones":[{"id":8}]}],"group":{"id":4,"name":"group","zones":[{"id":8}]}}`)
		case "GET /api/network-router-types":
			fmt.Fprint(w, `{"networkRouterTypes":[{"id":2,"code":"neutron","name":"Neutron"}]}`)
		case "GET /api/network-router-types/2":
			fmt.Fprint(w, `{"networkRouterType":{"id":2,"code":"neutron","name":"Neutron","creatable":true,"hasNetworkServer":false,"optionTypes":[]}}`)
		case "POST /api/networks/routers":
			fmt.Fprint(w, `{"success":true,"id":42}`)
		case "GET /api/networks/routers/42":
			if deleted {
				http.NotFound(w, r)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"networkRouter": map[string]any{"id": 42, "name": name, "enabled": enabled, "status": "available", "type": map[string]any{"id": 2, "code": "neutron", "name": "Neutron"}, "site": map[string]any{"id": 4, "name": "group"}, "zone": map[string]any{"id": 8}}})
		case "PUT /api/networks/routers/42":
			var body struct {
				Router struct {
					Name    string `json:"name"`
					Enabled bool   `json:"enabled"`
				} `json:"networkRouter"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			name = body.Router.Name
			enabled = body.Router.Enabled
			fmt.Fprint(w, `{"success":true}`)
		case "DELETE /api/networks/routers/42":
			deleted = true
			fmt.Fprint(w, `{"success":true}`)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	config := func(name string, enabled bool) string {
		return fmt.Sprintf(`
provider "mtncloud" {
 url = %q
 token = "test"
 group = "group"
}
data "mtncloud_router_types" "all" {}
resource "mtncloud_router" "app" {
 name = %q
 type = "neutron"
 enabled = %t
 options_json = jsonencode({config = {enableSnat = false}})
}
`, server.URL, name, enabled)
	}
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config("app", true), Check: resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr("mtncloud_router.app", "id", "42"), resource.TestCheckResourceAttr("mtncloud_router.app", "cloud_id", "8"), resource.TestCheckResourceAttr("mtncloud_router.app", "group", "group"))},
			{ResourceName: "mtncloud_router.app", ImportState: true, ImportStateVerify: true, ImportStateVerifyIgnore: []string{"options_json"}},
			{Config: config("renamed", false), Check: resource.TestCheckResourceAttr("mtncloud_router.app", "enabled", "false")},
		},
	})
}
