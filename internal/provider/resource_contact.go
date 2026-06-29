package provider

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mahveotm/terraform-provider-mtncloud/internal/client"
)

var _ resource.Resource = &contactResource{}
var _ resource.ResourceWithConfigure = &contactResource{}
var _ resource.ResourceWithImportState = &contactResource{}

type contactResource struct {
	resourceBase
}

type contactResourceModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	EmailAddress types.String `tfsdk:"email_address"`
	SMSAddress   types.String `tfsdk:"sms_address"`
	SlackHook    types.String `tfsdk:"slack_hook"`
}

func NewContactResource() resource.Resource { return &contactResource{} }

func (r *contactResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_contact"
}

func (r *contactResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = rschema.Schema{
		Description: "Manages an MTN Cloud monitoring contact: a notification target (email, SMS, " +
			"and/or Slack) that alerts can fan out to.",
		Attributes: map[string]rschema.Attribute{
			"id":            computedIDAttribute("Numeric identifier of the contact."),
			"name":          rschema.StringAttribute{Required: true, Description: "Name of the contact."},
			"email_address": rschema.StringAttribute{Optional: true, Description: "Email notification address."},
			"sms_address":   rschema.StringAttribute{Optional: true, Description: "SMS notification address."},
			"slack_hook":    rschema.StringAttribute{Optional: true, Description: "Slack incoming-webhook URL."},
		},
	}
}

func (r *contactResource) input(plan contactResourceModel) client.ContactInput {
	return client.ContactInput{
		Name:         plan.Name.ValueString(),
		EmailAddress: plan.EmailAddress.ValueString(),
		SMSAddress:   plan.SMSAddress.ValueString(),
		SlackHook:    plan.SlackHook.ValueString(),
	}
}

func (r *contactResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan contactResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	contact, err := r.client.CreateContact(ctx, r.input(plan))
	if err != nil {
		opError(&resp.Diagnostics, "Create", "Contact", err)
		return
	}
	setContactState(&plan, contact)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *contactResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state contactResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(state.ID, "Contact", &resp.Diagnostics)
	if !ok {
		return
	}
	contact, err := r.client.GetContact(ctx, id)
	if handleReadError(ctx, err, "Contact", &resp.State, &resp.Diagnostics) {
		return
	}
	setContactState(&state, contact)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *contactResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan contactResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(plan.ID, "Contact", &resp.Diagnostics)
	if !ok {
		return
	}
	contact, err := r.client.UpdateContact(ctx, id, r.input(plan))
	if err != nil {
		opError(&resp.Diagnostics, "Update", "Contact", err)
		return
	}
	setContactState(&plan, contact)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *contactResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state contactResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, ok := parseID(state.ID, "Contact", &resp.Diagnostics)
	if !ok {
		return
	}
	if err := r.client.DeleteContact(ctx, id); err != nil && !client.IsNotFound(err) {
		opError(&resp.Diagnostics, "Delete", "Contact", err)
	}
}

func (r *contactResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, pathRootID(), req, resp)
}

func setContactState(data *contactResourceModel, contact *client.Contact) {
	data.ID = types.StringValue(strconv.FormatInt(contact.ID, 10))
	data.Name = types.StringValue(contact.Name)
	data.EmailAddress = mergeAPIString(data.EmailAddress, contact.EmailAddress)
	data.SMSAddress = mergeAPIString(data.SMSAddress, contact.SMSAddress)
	data.SlackHook = mergeAPIString(data.SlackHook, contact.SlackHook)
}
