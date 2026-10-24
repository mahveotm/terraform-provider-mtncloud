package provider

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type routerTypesDataSource struct{ dataSourceBase }
type routerTypesModel struct {
	TypesJSON types.String `tfsdk:"types_json"`
}

func NewRouterTypesDataSource() datasource.DataSource { return &routerTypesDataSource{} }
func (d *routerTypesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_router_types"
}
func (d *routerTypesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dschema.Schema{Description: "Lists router types available to the authenticated account, including creation capabilities and type-specific option metadata. No name or code is required.", Attributes: map[string]dschema.Attribute{
		"types_json": dschema.StringAttribute{Computed: true, Description: "JSON array of router types with id, name, code, creatable, hasNetworkServer, and optionTypes. Use jsondecode to inspect the options returned by Morpheus."},
	}}
}
func (d *routerTypesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	routerTypes, err := d.client.ListRouterTypes(ctx)
	if err != nil {
		resp.Diagnostics.AddError("List Router Types Failed", err.Error())
		return
	}
	// Fetch details because list responses may not include complete option metadata.
	for i := range routerTypes {
		detail, err := d.client.GetRouterType(ctx, routerTypes[i].ID)
		if err != nil {
			resp.Diagnostics.AddError("Read Router Type Failed", err.Error())
			return
		}
		routerTypes[i] = *detail
	}
	raw, err := json.Marshal(routerTypes)
	if err != nil {
		resp.Diagnostics.AddError("Encode Router Types Failed", err.Error())
		return
	}
	if string(raw) == "null" {
		raw = []byte("[]")
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &routerTypesModel{TypesJSON: types.StringValue(string(raw))})...)
}
