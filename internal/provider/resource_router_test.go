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
// including import and refresh. No live MTN infrastructure is used.
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
			fmt.Fprint(w, `{"networkRouterTypes":[{"id":2,"code":"neutronRouter","name":"Neutron Router"}]}`)
		case "GET /api/network-router-types/2":
			fmt.Fprint(w, `{"networkRouterType":{"id":2,"code":"neutronRouter","name":"Neutron Router","creatable":true,"hasNetworkServer":false,"optionTypes":[]}}`)
		case "GET /api/options/zonePools":
			fmt.Fprint(w, `{"data":[{"id":208,"name":"project","value":"pool-208"}]}`)
		case "GET /api/networks":
			if r.URL.Query().Get("zoneId") != "8" {
				t.Error("network lookup must be scoped to cloud 8")
			}
			if r.URL.Query().Get("name") == "public-network-01" {
				fmt.Fprint(w, `{"networks":[{"id":34,"name":"public-network-01"}]}`)
			} else {
				fmt.Fprint(w, `{"networks":[{"id":7,"name":"app-net"}]}`)
			}
		case "GET /api/networks/34":
			fmt.Fprint(w, `{"network":{"id":34,"name":"public-network-01","zone":{"id":8},"subnets":[]}}`)
		case "GET /api/networks/7":
			fmt.Fprint(w, `{"network":{"id":7,"name":"app-net","zone":{"id":8},"zonePool":{"id":208,"name":"project"},"subnets":[{"id":9,"cidr":"10.1.0.0/24"}]}}`)
		case "POST /api/networks/routers":
			var payload struct {
				Router map[string]json.RawMessage `json:"networkRouter"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Error(err)
			}
			if string(payload.Router["poolId"]) != `"pool-208"` || string(payload.Router["config"]) != `{"poolId":"pool-208"}` || string(payload.Router["type"]) != `{"id":2}` {
				t.Errorf("unexpected MTN router payload: %s", payload.Router)
			}
			if string(payload.Router["interfaces"]) != `[{"networkSubnet":{"id":9}}]` || string(payload.Router["externalNetwork"]) != `{"id":34}` {
				t.Errorf("router must use discovered network and subnet IDs: %s", payload.Router)
			}
			fmt.Fprint(w, `{"success":true,"id":42}`)
		case "GET /api/networks/routers/42":
			if deleted {
				http.NotFound(w, r)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"networkRouter": map[string]any{"id": 42, "name": name, "enabled": enabled, "status": "available", "type": map[string]any{"id": 2, "code": "neutronRouter", "name": "Neutron Router"}, "site": map[string]any{"id": 4, "name": "group"}, "zone": map[string]any{"id": 8}, "externalNetwork": map[string]any{"id": 34}, "interfaces": []map[string]any{{"cidr": "10.1.0.0/24", "network": map[string]any{"id": 7}}}}})
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
 resource_pool = "pool-208"
}

data "mtncloud_network" "app" {
 name = "app-net"
 cloud_id = 8
}

data "mtncloud_network" "public" {
 name = "public-network-01"
 cloud_id = 8
}

resource "mtncloud_router" "app" {
 name = %q
 internal_subnet_ids = data.mtncloud_network.app.subnet_ids
 external_network_id = tonumber(data.mtncloud_network.public.id)
 enabled = %t

}
`, server.URL, name, enabled)
	}
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config("app", true), Check: resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr("mtncloud_router.app", "id", "42"), resource.TestCheckResourceAttr("mtncloud_router.app", "cloud_id", "8"), resource.TestCheckResourceAttr("mtncloud_router.app", "group", "group"),
				resource.TestCheckResourceAttr("data.mtncloud_network.app", "subnet_ids.#", "1"),
				resource.TestCheckTypeSetElemAttr("data.mtncloud_network.app", "subnet_ids.*", "9"),
				resource.TestCheckResourceAttr("data.mtncloud_network.public", "subnet_ids.#", "0"))},
			{ResourceName: "mtncloud_router.app", ImportState: true, ImportStateVerify: true},
		},
	})
}
