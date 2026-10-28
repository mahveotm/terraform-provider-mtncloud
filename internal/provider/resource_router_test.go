package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Exercise Terraform's plan/state contract against the documented API shapes,
// including import and refresh. No live MTN infrastructure is used.
func TestRouterLifecycleWithMockAPI(t *testing.T) {
	t.Setenv("MTN_CLOUD_USERNAME", "")
	t.Setenv("MTN_CLOUD_PASSWORD", "")
	externalOptions, err := os.ReadFile("testdata/external_network_options.json")
	if err != nil {
		t.Fatal(err)
	}
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
			if r.URL.Query().Get("name") == "app-net" {
				fmt.Fprint(w, `{"networks":[{"id":7,"name":"app-net"}]}`)
			} else {
				fmt.Fprint(w, `{"networks":[]}`)
			}
		case "GET /api/options/openstackExternalNetworks":
			query := r.URL.Query()
			for key, value := range map[string]string{"router.zone.id": "8", "router.site.id": "4", "router.poolId": "pool-208", "routerType.id": "2"} {
				if query.Get(key) != value {
					t.Errorf("external network query %s = %q, want %q", key, query.Get(key), value)
				}
			}
			_, _ = w.Write(externalOptions)
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

data "mtncloud_external_network" "public" {
 name = "public-network-01"
}

data "mtncloud_external_networks" "available" {}

resource "mtncloud_router" "app" {
 name = %q
 internal_subnet_ids = data.mtncloud_network.app.subnet_ids
 external_network_id = tonumber(data.mtncloud_external_network.public.id)
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
				resource.TestCheckResourceAttr("data.mtncloud_external_network.public", "id", "34"),
				resource.TestCheckResourceAttr("data.mtncloud_external_networks.available", "networks.#", "4"))},
			{ResourceName: "mtncloud_router.app", ImportState: true, ImportStateVerify: true},
			{Config: strings.Replace(config("app", true), `name = "public-network-01"`, `name = "missing-network"`, 1), ExpectError: regexp.MustCompile("MTN External Network Not Found")},
		},
	})
}
