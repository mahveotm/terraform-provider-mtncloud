package client

import (
	"context"
	"fmt"
)

// ExecuteSchedule is a cron schedule used to trigger jobs. It lives at
// /execute-schedules (NOT /schedules, which 404s on MTN); the single-object
// wrapper is "schedule" and the collection wrapper is "schedules". scheduleType
// is fixed to "execute".
type ExecuteSchedule struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Enabled      *bool  `json:"enabled"`
	Cron         string `json:"cron"`
	Timezone     string `json:"scheduleTimezone"`
	Visibility   string `json:"visibility"`
	ScheduleType string `json:"scheduleType"`
}

// ExecuteScheduleInput is the create/update payload.
type ExecuteScheduleInput struct {
	Name        string
	Description string
	Enabled     *bool
	Cron        string
	Timezone    string
	Visibility  string
}

func executeScheduleBody(in ExecuteScheduleInput) map[string]any {
	s := map[string]any{
		"name":         in.Name,
		"scheduleType": "execute",
	}
	if in.Description != "" {
		s["description"] = in.Description
	}
	if in.Enabled != nil {
		s["enabled"] = *in.Enabled
	}
	if in.Cron != "" {
		s["cron"] = in.Cron
	}
	if in.Timezone != "" {
		s["scheduleTimezone"] = in.Timezone
	}
	if in.Visibility != "" {
		s["visibility"] = in.Visibility
	}
	return map[string]any{"schedule": s}
}

func (c *Client) CreateExecuteSchedule(ctx context.Context, input ExecuteScheduleInput) (*ExecuteSchedule, error) {
	return createObj[ExecuteSchedule](c, ctx, "/execute-schedules", "schedule", executeScheduleBody(input))
}

func (c *Client) GetExecuteSchedule(ctx context.Context, id int64) (*ExecuteSchedule, error) {
	return getByID[ExecuteSchedule](c, ctx, fmt.Sprintf("/execute-schedules/%d", id), "schedule")
}

func (c *Client) GetExecuteScheduleByName(ctx context.Context, name string) (*ExecuteSchedule, error) {
	return firstByName[ExecuteSchedule](c, ctx, "/execute-schedules", "schedules", name)
}

func (c *Client) UpdateExecuteSchedule(ctx context.Context, id int64, input ExecuteScheduleInput) (*ExecuteSchedule, error) {
	return updateObj[ExecuteSchedule](c, ctx, fmt.Sprintf("/execute-schedules/%d", id), "schedule", executeScheduleBody(input))
}

func (c *Client) DeleteExecuteSchedule(ctx context.Context, id int64) error {
	return c.delete(ctx, fmt.Sprintf("/execute-schedules/%d", id), nil)
}

func (c *Client) ListExecuteSchedules(ctx context.Context) ([]ExecuteSchedule, error) {
	return listObjects[ExecuteSchedule](c, ctx, "/execute-schedules", "schedules")
}
