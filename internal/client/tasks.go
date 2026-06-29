package client

import (
	"context"
	"fmt"
	"sort"
)

// Task is an automation task in MTN Cloud. All task types live at /tasks and are
// discriminated by taskType.code; the Terraform layer exposes a friendly `type`
// (shell, python, …) mapped to that code. Only stable metadata round-trips on
// read; type-specific options and script content are config-authoritative.
type Task struct {
	ID                int64  `json:"id"`
	Name              string `json:"name"`
	Code              string `json:"code"`
	ExecuteTarget     string `json:"executeTarget"`
	ResultType        string `json:"resultType"`
	Retryable         *bool  `json:"retryable"`
	RetryCount        *int64 `json:"retryCount"`
	RetryDelaySeconds *int64 `json:"retryDelaySeconds"`
	ContinueOnError   *bool  `json:"continueOnError"`
	AllowCustomConfig *bool  `json:"allowCustomConfig"`
	Visibility        string `json:"visibility"`
	TaskType          struct {
		Code string `json:"code"`
		Name string `json:"name"`
	} `json:"taskType"`
}

// taskTypeCodes maps the friendly Terraform `type` to the Morpheus taskType.code.
var taskTypeCodes = map[string]string{
	"shell":      "script",
	"python":     "jythonTask",
	"ansible":    "ansibleTask",
	"powershell": "winrmTask",
	"email":      "email",
	"restart":    "restartTask",
}

// TaskTypes is the sorted list of friendly task types accepted by the provider.
var TaskTypes = func() []string {
	out := make([]string, 0, len(taskTypeCodes))
	for k := range taskTypeCodes {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}()

// TaskTypeFromCode reverses taskTypeCodes (Morpheus code -> friendly type), used
// on read/import to populate `type` from the API's taskType.code.
func TaskTypeFromCode(code string) string {
	for friendly, c := range taskTypeCodes {
		if c == code {
			return friendly
		}
	}
	return ""
}

// TaskInput carries the union of fields across task types. Which ones are used
// depends on Type; mapTaskPayload selects and shapes them.
type TaskInput struct {
	Type              string
	Name              string
	Code              string
	Labels            []string
	ExecuteTarget     string
	ResultType        string
	Retryable         *bool
	RetryCount        *int64
	RetryDelaySeconds *int64
	ContinueOnError   *bool
	AllowCustomConfig *bool
	Visibility        string
	CredentialID      *int64

	// Source (script + email types) -> API "file" object.
	SourceType   string
	Content      string
	ContentPath  string
	ContentRef   string
	RepositoryID *int64

	// shell
	Sudo bool
	// remote exec (shell/powershell)
	Host     string
	Port     string
	Username string
	Password string
	// powershell
	Elevated bool
	// python
	PythonBinary             string
	PythonArgs               string
	PythonAdditionalPackages string
	// ansible
	GitID    *int64
	GitRef   string
	Playbook string
	Tags     string
	SkipTags string
	Options  string
	// email
	EmailAddress        string
	Subject             string
	SkipWrappedTemplate bool
}

func mapTaskPayload(in TaskInput) map[string]any {
	task := map[string]any{
		"name":     in.Name,
		"taskType": map[string]any{"code": taskTypeCodes[in.Type]},
	}
	if in.Code != "" {
		task["code"] = in.Code
	}
	if in.Labels != nil {
		task["labels"] = in.Labels
	}
	if in.ExecuteTarget != "" {
		task["executeTarget"] = in.ExecuteTarget
	}
	if in.ResultType != "" {
		task["resultType"] = in.ResultType
	}
	if in.Retryable != nil {
		task["retryable"] = *in.Retryable
	}
	if in.RetryCount != nil {
		task["retryCount"] = *in.RetryCount
	}
	if in.RetryDelaySeconds != nil {
		task["retryDelaySeconds"] = *in.RetryDelaySeconds
	}
	if in.ContinueOnError != nil {
		task["continueOnError"] = *in.ContinueOnError
	}
	if in.AllowCustomConfig != nil {
		task["allowCustomConfig"] = *in.AllowCustomConfig
	}
	if in.Visibility != "" {
		task["visibility"] = in.Visibility
	}
	if in.CredentialID != nil {
		task["credential"] = map[string]any{"id": *in.CredentialID}
	}

	if in.SourceType != "" {
		file := map[string]any{"sourceType": in.SourceType}
		switch in.SourceType {
		case "local":
			file["content"] = in.Content
		case "url":
			file["contentPath"] = in.ContentPath
		case "repository":
			file["contentPath"] = in.ContentPath
			if in.ContentRef != "" {
				file["contentRef"] = in.ContentRef
			}
			if in.RepositoryID != nil {
				file["repository"] = map[string]any{"id": *in.RepositoryID}
			}
		}
		task["file"] = file
	}

	opts := map[string]any{}
	switch in.Type {
	case "shell":
		if in.Sudo {
			opts["shell.sudo"] = "on"
		}
		addRemote(opts, in)
	case "powershell":
		if in.Elevated {
			opts["winrm.elevated"] = "on"
		}
		addRemote(opts, in)
	case "python":
		setIf(opts, "pythonBinary", in.PythonBinary)
		setIf(opts, "pythonArgs", in.PythonArgs)
		setIf(opts, "pythonAdditionalPackages", in.PythonAdditionalPackages)
	case "ansible":
		if in.GitID != nil {
			opts["ansibleGitId"] = *in.GitID
		}
		setIf(opts, "ansibleGitRef", in.GitRef)
		setIf(opts, "ansiblePlaybook", in.Playbook)
		setIf(opts, "ansibleTags", in.Tags)
		setIf(opts, "ansibleSkipTags", in.SkipTags)
		setIf(opts, "ansibleOptions", in.Options)
	case "email":
		setIf(opts, "emailAddress", in.EmailAddress)
		setIf(opts, "emailSubject", in.Subject)
		if in.SkipWrappedTemplate {
			opts["emailSkipTemplate"] = "on"
		}
	}
	if len(opts) > 0 {
		task["taskOptions"] = opts
	}

	return map[string]any{"task": task}
}

func addRemote(opts map[string]any, in TaskInput) {
	if in.ExecuteTarget != "remote" {
		return
	}
	setIf(opts, "host", in.Host)
	setIf(opts, "port", in.Port)
	setIf(opts, "username", in.Username)
	setIf(opts, "password", in.Password)
}

func setIf(m map[string]any, key, value string) {
	if value != "" {
		m[key] = value
	}
}

func (c *Client) CreateTask(ctx context.Context, input TaskInput) (*Task, error) {
	return createObj[Task](c, ctx, "/tasks", "task", mapTaskPayload(input))
}

func (c *Client) GetTask(ctx context.Context, id int64) (*Task, error) {
	return getByID[Task](c, ctx, fmt.Sprintf("/tasks/%d", id), "task")
}

func (c *Client) GetTaskByName(ctx context.Context, name string) (*Task, error) {
	return firstByName[Task](c, ctx, "/tasks", "tasks", name)
}

func (c *Client) UpdateTask(ctx context.Context, id int64, input TaskInput) (*Task, error) {
	return updateObj[Task](c, ctx, fmt.Sprintf("/tasks/%d", id), "task", mapTaskPayload(input))
}

func (c *Client) DeleteTask(ctx context.Context, id int64) error {
	return c.delete(ctx, fmt.Sprintf("/tasks/%d", id), nil)
}

func (c *Client) ListTasks(ctx context.Context) ([]Task, error) {
	return listObjects[Task](c, ctx, "/tasks", "tasks")
}
