package provider

import (
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

// computedIDAttribute returns the standard Computed numeric "id" attribute used
// by every resource. UseStateForUnknown pins the value across in-place updates;
// without it a Computed id plans as "known after apply", so plan.ID is unknown
// during Update and surfaces as `Could not parse "" as a numeric ID`.
func computedIDAttribute(description string) rschema.StringAttribute {
	return rschema.StringAttribute{
		Computed:      true,
		Description:   description,
		PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
	}
}
