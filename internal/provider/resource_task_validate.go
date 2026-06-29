package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// nullable is satisfied by every framework value type (types.String/Bool/Int64/…)
// and lets ValidateConfig treat heterogeneous attributes uniformly.
type nullable interface {
	IsNull() bool
	IsUnknown() bool
}

func attrSet(v nullable) bool { return !v.IsNull() && !v.IsUnknown() }

func attrPresent(v nullable) bool { return !v.IsNull() }

func containsStr(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}

// ValidateConfig enforces per-type validity for the single mtncloud_task resource:
// fields that don't belong to the selected `type` are rejected, and the fields a
// type requires are checked present. This gives per-type resources' strictness
// without separate resources (the AWS ConflictsWith/CustomizeDiff approach).
func (r *taskResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var cfg taskResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() || !attrSet(cfg.Type) {
		return
	}
	tt := cfg.Type.ValueString()

	scriptOrEmail := []string{"shell", "python", "powershell", "email"}
	scriptTypes := []string{"shell", "python", "powershell"}
	remoteTypes := []string{"shell", "powershell"}

	specific := []struct {
		name    string
		value   nullable
		allowed []string
	}{
		{"source_type", cfg.SourceType, scriptOrEmail},
		{"content", cfg.Content, scriptOrEmail},
		{"content_path", cfg.ContentPath, scriptOrEmail},
		{"content_ref", cfg.ContentRef, scriptOrEmail},
		{"repository_id", cfg.RepositoryID, scriptOrEmail},
		{"result_type", cfg.ResultType, scriptTypes},
		{"sudo", cfg.Sudo, []string{"shell"}},
		{"elevated", cfg.Elevated, []string{"powershell"}},
		{"host", cfg.Host, remoteTypes},
		{"port", cfg.Port, remoteTypes},
		{"username", cfg.Username, remoteTypes},
		{"password", cfg.Password, remoteTypes},
		{"python_binary", cfg.PythonBinary, []string{"python"}},
		{"python_args", cfg.PythonArgs, []string{"python"}},
		{"python_additional_packages", cfg.PythonAdditionalPackages, []string{"python"}},
		{"git_id", cfg.GitID, []string{"ansible"}},
		{"git_ref", cfg.GitRef, []string{"ansible"}},
		{"playbook", cfg.Playbook, []string{"ansible"}},
		{"tags", cfg.Tags, []string{"ansible"}},
		{"skip_tags", cfg.SkipTags, []string{"ansible"}},
		{"options", cfg.Options, []string{"ansible"}},
		{"email_address", cfg.EmailAddress, []string{"email"}},
		{"subject", cfg.Subject, []string{"email"}},
		{"skip_wrapped_template", cfg.SkipWrappedTemplate, []string{"email"}},
	}
	for _, a := range specific {
		if attrPresent(a.value) && !containsStr(a.allowed, tt) {
			resp.Diagnostics.AddAttributeError(path.Root(a.name), "Attribute Not Valid For Task Type",
				fmt.Sprintf("`%s` is not valid when `type = %q`; it applies to: %s.", a.name, tt, joinQuoted(a.allowed)))
		}
	}

	require := func(name string, v nullable) {
		if !attrPresent(v) {
			resp.Diagnostics.AddAttributeError(path.Root(name), "Missing Required Attribute",
				fmt.Sprintf("`%s` is required when `type = %q`.", name, tt))
		}
	}
	switch tt {
	case "shell", "python", "powershell":
		require("source_type", cfg.SourceType)
	case "email":
		require("source_type", cfg.SourceType)
		require("email_address", cfg.EmailAddress)
	case "ansible":
		require("playbook", cfg.Playbook)
	}

	if attrSet(cfg.SourceType) {
		switch cfg.SourceType.ValueString() {
		case "local":
			require("content", cfg.Content)
		case "url", "repository":
			require("content_path", cfg.ContentPath)
		}
	}
}
