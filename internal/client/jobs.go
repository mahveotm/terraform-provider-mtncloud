package client

import (
	"context"
	"fmt"
	"strconv"
)

// Job runs a workflow or a task on a schedule against targets. It lives at /jobs.
// Exactly one of Workflow/Task is set. scheduleMode is encoded from a friendly
// value: "manual" -> "manual", "date_time" -> "dateTime" (+dateTime), "schedule"
// -> the execute-schedule id as a string.
type Job struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	Enabled       *bool   `json:"enabled"`
	Workflow      *jobRef `json:"workflow"`
	Task          *jobRef `json:"task"`
	TargetType    string  `json:"targetType"`
	InstanceLabel string  `json:"instanceLabel"`
}

type jobRef struct {
	ID int64 `json:"id"`
}

// JobScheduleModes are the friendly schedule modes the provider accepts.
var JobScheduleModes = []string{"manual", "date_time", "schedule"}

// JobTargetTypes are the target types supported on MTN (server/server-label ride
// on /servers, which is 403 on the tenant token, so they are intentionally omitted).
var JobTargetTypes = []string{"instance", "instance-label", "appliance"}

// JobInput is the create/update payload.
type JobInput struct {
	Name              string
	Labels            []string
	Enabled           *bool
	WorkflowID        *int64
	TaskID            *int64
	ScheduleMode      string
	DateTime          string
	ExecuteScheduleID *int64
	TargetType        string
	Targets           []int64
	InstanceLabel     string
	CustomOptions     map[string]string
	CustomConfig      string
}

func jobBody(in JobInput) map[string]any {
	job := map[string]any{"name": in.Name}
	if in.Labels != nil {
		job["labels"] = in.Labels
	}
	if in.Enabled != nil {
		job["enabled"] = *in.Enabled
	}
	if in.WorkflowID != nil {
		job["workflow"] = map[string]any{"id": *in.WorkflowID}
	}
	if in.TaskID != nil {
		job["task"] = map[string]any{"id": *in.TaskID}
	}
	switch in.ScheduleMode {
	case "manual":
		job["scheduleMode"] = "manual"
	case "date_time":
		job["scheduleMode"] = "dateTime"
		if in.DateTime != "" {
			job["dateTime"] = in.DateTime
		}
	case "schedule":
		if in.ExecuteScheduleID != nil {
			job["scheduleMode"] = strconv.FormatInt(*in.ExecuteScheduleID, 10)
		}
	}
	if in.TargetType != "" {
		job["targetType"] = in.TargetType
	}
	if in.InstanceLabel != "" {
		job["instanceLabel"] = in.InstanceLabel
	}
	if len(in.Targets) > 0 {
		targets := make([]map[string]any, 0, len(in.Targets))
		for _, id := range in.Targets {
			targets = append(targets, map[string]any{"refId": id})
		}
		job["targets"] = targets
	}
	if len(in.CustomOptions) > 0 {
		job["customOptions"] = in.CustomOptions
	}
	if in.CustomConfig != "" {
		job["customConfig"] = in.CustomConfig
	}
	return map[string]any{"job": job}
}

func (c *Client) CreateJob(ctx context.Context, input JobInput) (*Job, error) {
	return createObj[Job](c, ctx, "/jobs", "job", jobBody(input))
}

func (c *Client) GetJob(ctx context.Context, id int64) (*Job, error) {
	return getByID[Job](c, ctx, fmt.Sprintf("/jobs/%d", id), "job")
}

func (c *Client) GetJobByName(ctx context.Context, name string) (*Job, error) {
	return firstByName[Job](c, ctx, "/jobs", "jobs", name)
}

func (c *Client) UpdateJob(ctx context.Context, id int64, input JobInput) (*Job, error) {
	return updateObj[Job](c, ctx, fmt.Sprintf("/jobs/%d", id), "job", jobBody(input))
}

func (c *Client) DeleteJob(ctx context.Context, id int64) error {
	return c.delete(ctx, fmt.Sprintf("/jobs/%d", id), nil)
}

func (c *Client) ListJobs(ctx context.Context) ([]Job, error) {
	return listObjects[Job](c, ctx, "/jobs", "jobs")
}
