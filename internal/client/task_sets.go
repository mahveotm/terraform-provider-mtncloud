package client

import (
	"context"
	"fmt"
)

// Workflow is an MTN Cloud task-set: an ordered set of tasks run together, either
// as an operational workflow (type "operation") or a provisioning workflow
// (type "provision"). It lives at /task-sets.
type Workflow struct {
	ID                int64         `json:"id"`
	Name              string        `json:"name"`
	Description       string        `json:"description"`
	Type              string        `json:"type"`
	Visibility        string        `json:"visibility"`
	Platform          string        `json:"platform"`
	AllowCustomConfig *bool         `json:"allowCustomConfig"`
	TaskSetTasks      []TaskSetTask `json:"taskSetTasks"`
}

// TaskSetTask is a member task within a workflow, with its phase and order.
type TaskSetTask struct {
	Task struct {
		ID int64 `json:"id"`
	} `json:"task"`
	TaskPhase string `json:"taskPhase"`
	TaskOrder int64  `json:"taskOrder"`
}

// WorkflowTypes are the friendly/API task-set types (they match the API verbatim).
var WorkflowTypes = []string{"operation", "provision"}

// WorkflowPlatforms are the accepted platform filters.
var WorkflowPlatforms = []string{"all", "linux", "macos", "windows"}

// OperationalPhase is the single phase used by operational workflows.
const OperationalPhase = "operation"

// ProvisionPhases are the phases a provisioning workflow's tasks may run in.
var ProvisionPhases = []string{
	"configure", "preProvision", "provision", "postProvision",
	"start", "stop", "preDeploy", "deploy", "reconfigure",
	"teardown", "shutdown", "startup",
}

// WorkflowTask is a member task reference in the create/update payload.
type WorkflowTask struct {
	TaskID int64
	Phase  string
}

// WorkflowInput is the create/update payload.
type WorkflowInput struct {
	Name              string
	Description       string
	Type              string
	Labels            []string
	Visibility        string
	Platform          string
	AllowCustomConfig *bool
	Tasks             []WorkflowTask
}

func taskSetBody(in WorkflowInput) map[string]any {
	ts := map[string]any{
		"name": in.Name,
		"type": in.Type,
	}
	if in.Description != "" {
		ts["description"] = in.Description
	}
	if in.Labels != nil {
		ts["labels"] = in.Labels
	}
	if in.Visibility != "" {
		ts["visibility"] = in.Visibility
	}
	if in.Platform != "" {
		ts["platform"] = in.Platform
	}
	if in.AllowCustomConfig != nil {
		ts["allowCustomConfig"] = *in.AllowCustomConfig
	}
	tasks := make([]map[string]any, 0, len(in.Tasks))
	for _, t := range in.Tasks {
		tasks = append(tasks, map[string]any{"taskId": t.TaskID, "taskPhase": t.Phase})
	}
	ts["tasks"] = tasks
	return map[string]any{"taskSet": ts}
}

func (c *Client) CreateWorkflow(ctx context.Context, input WorkflowInput) (*Workflow, error) {
	return createObj[Workflow](c, ctx, "/task-sets", "taskSet", taskSetBody(input))
}

func (c *Client) GetWorkflow(ctx context.Context, id int64) (*Workflow, error) {
	return getByID[Workflow](c, ctx, fmt.Sprintf("/task-sets/%d", id), "taskSet")
}

func (c *Client) GetWorkflowByName(ctx context.Context, name string) (*Workflow, error) {
	return firstByName[Workflow](c, ctx, "/task-sets", "taskSets", name)
}

func (c *Client) UpdateWorkflow(ctx context.Context, id int64, input WorkflowInput) (*Workflow, error) {
	return updateObj[Workflow](c, ctx, fmt.Sprintf("/task-sets/%d", id), "taskSet", taskSetBody(input))
}

func (c *Client) DeleteWorkflow(ctx context.Context, id int64) error {
	return c.delete(ctx, fmt.Sprintf("/task-sets/%d", id), nil)
}

func (c *Client) ListWorkflows(ctx context.Context) ([]Workflow, error) {
	return listObjects[Workflow](c, ctx, "/task-sets", "taskSets")
}
