package client

import (
	"context"
	"fmt"
)

// Monitoring covers the observability surface of MTN Cloud (Morpheus): contacts
// (notification targets), checks (individual monitors), check groups (collections
// of checks rolled up to one health status), and alerts (notification rules that
// fan checks/groups out to contacts). All four live under /api/monitoring and are
// writable with the Customer-Admin token.

// idRef is the {"id": N} shape Morpheus uses inside relational arrays.
type idRef struct {
	ID int64 `json:"id"`
}

func refIDs(items []idRef) []int64 {
	out := make([]int64, 0, len(items))
	for _, item := range items {
		out = append(out, item.ID)
	}
	return out
}

// idObjects turns a slice of numeric IDs into the [{"id": N}] array shape some
// Morpheus relational fields expect (e.g. alert contacts).
func idObjects(ids []int64) []map[string]any {
	out := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		out = append(out, map[string]any{"id": id})
	}
	return out
}

// ----- Monitoring Contact -----

// Contact is a monitoring notification target (email / SMS / Slack).
type Contact struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	EmailAddress string `json:"emailAddress"`
	SMSAddress   string `json:"smsAddress"`
	SlackHook    string `json:"slackHook"`
}

// ContactInput is the create/update payload.
type ContactInput struct {
	Name         string
	EmailAddress string
	SMSAddress   string
	SlackHook    string
}

// contactBody builds the create/update payload. On update (clearable=true) the
// channel fields are always sent — including empty strings — so removing one from
// config actually clears it remotely; on create they are omitted when empty.
// Mirrors the mapRulePayload(input, update) convention used for security-group rules.
func contactBody(input ContactInput, clearable bool) map[string]any {
	contact := map[string]any{"name": input.Name}
	put := func(key, value string) {
		if value != "" || clearable {
			contact[key] = value
		}
	}
	put("emailAddress", input.EmailAddress)
	put("smsAddress", input.SMSAddress)
	put("slackHook", input.SlackHook)
	return map[string]any{"contact": contact}
}

func (c *Client) CreateContact(ctx context.Context, input ContactInput) (*Contact, error) {
	return createObj[Contact](c, ctx, "/monitoring/contacts", "contact", contactBody(input, false))
}

func (c *Client) GetContact(ctx context.Context, id int64) (*Contact, error) {
	return getByID[Contact](c, ctx, fmt.Sprintf("/monitoring/contacts/%d", id), "contact")
}

func (c *Client) GetContactByName(ctx context.Context, name string) (*Contact, error) {
	return firstByName[Contact](c, ctx, "/monitoring/contacts", "contacts", name)
}

func (c *Client) UpdateContact(ctx context.Context, id int64, input ContactInput) (*Contact, error) {
	return updateObj[Contact](c, ctx, fmt.Sprintf("/monitoring/contacts/%d", id), "contact", contactBody(input, true))
}

func (c *Client) DeleteContact(ctx context.Context, id int64) error {
	return c.delete(ctx, fmt.Sprintf("/monitoring/contacts/%d", id), nil)
}

func (c *Client) ListContacts(ctx context.Context) ([]Contact, error) {
	return listObjects[Contact](c, ctx, "/monitoring/contacts", "contacts")
}

// ----- Monitoring Check -----

// Check is an individual monitor. checkType is immutable (it selects the monitor
// implementation); config carries the per-type settings as a JSON passthrough.
type Check struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	CheckInterval *int64 `json:"checkInterval"`
	InUptime      *bool  `json:"inUptime"`
	Active        *bool  `json:"active"`
	Severity      string `json:"severity"`
	CheckType     struct {
		Code string `json:"code"`
		Name string `json:"name"`
	} `json:"checkType"`
}

// CheckInput is the create/update payload. CheckType is the check-type code
// (e.g. "webGetCheck"); Config is the per-type config object (may be nil).
type CheckInput struct {
	Name          string
	CheckType     string
	Description   string
	CheckInterval *int64
	InUptime      *bool
	Active        *bool
	Severity      string
	Config        map[string]any
}

func checkBody(input CheckInput) map[string]any {
	check := map[string]any{"name": input.Name}
	if input.CheckType != "" {
		check["checkType"] = map[string]any{"code": input.CheckType}
	}
	if input.Description != "" {
		check["description"] = input.Description
	}
	if input.CheckInterval != nil {
		check["checkInterval"] = *input.CheckInterval
	}
	if input.InUptime != nil {
		check["inUptime"] = *input.InUptime
	}
	if input.Active != nil {
		check["active"] = *input.Active
	}
	if input.Severity != "" {
		check["severity"] = input.Severity
	}
	if len(input.Config) > 0 {
		check["config"] = input.Config
	}
	return map[string]any{"check": check}
}

func (c *Client) CreateCheck(ctx context.Context, input CheckInput) (*Check, error) {
	return createObj[Check](c, ctx, "/monitoring/checks", "check", checkBody(input))
}

func (c *Client) GetCheck(ctx context.Context, id int64) (*Check, error) {
	return getByID[Check](c, ctx, fmt.Sprintf("/monitoring/checks/%d", id), "check")
}

func (c *Client) GetCheckByName(ctx context.Context, name string) (*Check, error) {
	return firstByName[Check](c, ctx, "/monitoring/checks", "checks", name)
}

func (c *Client) UpdateCheck(ctx context.Context, id int64, input CheckInput) (*Check, error) {
	return updateObj[Check](c, ctx, fmt.Sprintf("/monitoring/checks/%d", id), "check", checkBody(input))
}

func (c *Client) DeleteCheck(ctx context.Context, id int64) error {
	return c.delete(ctx, fmt.Sprintf("/monitoring/checks/%d", id), nil)
}

func (c *Client) ListChecks(ctx context.Context) ([]Check, error) {
	return listObjects[Check](c, ctx, "/monitoring/checks", "checks")
}

// ----- Monitoring Check Group -----

// MonitoringGroup is a check group: several checks rolled up to a single health
// status (healthy while at least minHappy members are happy). The API path is
// /monitoring/groups but the JSON envelope key is "checkGroup"/"checkGroups".
type MonitoringGroup struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	MinHappy    *int64  `json:"minHappy"`
	InUptime    *bool   `json:"inUptime"`
	Severity    string  `json:"severity"`
	Active      *bool   `json:"active"`
	Checks      []idRef `json:"checks"`
}

func (g *MonitoringGroup) CheckIDs() []int64 { return refIDs(g.Checks) }

// MonitoringGroupInput is the create/update payload.
type MonitoringGroupInput struct {
	Name        string
	Description string
	MinHappy    *int64
	InUptime    *bool
	Severity    string
	Active      *bool
	CheckIDs    []int64
}

func monitoringGroupBody(input MonitoringGroupInput) map[string]any {
	group := map[string]any{"name": input.Name}
	if input.Description != "" {
		group["description"] = input.Description
	}
	if input.MinHappy != nil {
		group["minHappy"] = *input.MinHappy
	}
	if input.InUptime != nil {
		group["inUptime"] = *input.InUptime
	}
	if input.Severity != "" {
		group["severity"] = input.Severity
	}
	if input.Active != nil {
		group["active"] = *input.Active
	}
	if input.CheckIDs != nil {
		group["checks"] = input.CheckIDs
	}
	return map[string]any{"checkGroup": group}
}

func (c *Client) CreateMonitoringGroup(ctx context.Context, input MonitoringGroupInput) (*MonitoringGroup, error) {
	return createObj[MonitoringGroup](c, ctx, "/monitoring/groups", "checkGroup", monitoringGroupBody(input))
}

func (c *Client) GetMonitoringGroup(ctx context.Context, id int64) (*MonitoringGroup, error) {
	return getByID[MonitoringGroup](c, ctx, fmt.Sprintf("/monitoring/groups/%d", id), "checkGroup")
}

func (c *Client) GetMonitoringGroupByName(ctx context.Context, name string) (*MonitoringGroup, error) {
	return firstByName[MonitoringGroup](c, ctx, "/monitoring/groups", "checkGroups", name)
}

func (c *Client) UpdateMonitoringGroup(ctx context.Context, id int64, input MonitoringGroupInput) (*MonitoringGroup, error) {
	return updateObj[MonitoringGroup](c, ctx, fmt.Sprintf("/monitoring/groups/%d", id), "checkGroup", monitoringGroupBody(input))
}

func (c *Client) DeleteMonitoringGroup(ctx context.Context, id int64) error {
	return c.delete(ctx, fmt.Sprintf("/monitoring/groups/%d", id), nil)
}

func (c *Client) ListMonitoringGroups(ctx context.Context) ([]MonitoringGroup, error) {
	return listObjects[MonitoringGroup](c, ctx, "/monitoring/groups", "checkGroups")
}

// ----- Monitoring Alert -----

// Alert is a notification rule: when any referenced check/group/app crosses
// minSeverity (or all checks, when allChecks is set), the listed contacts are
// notified.
type Alert struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	MinSeverity string  `json:"minSeverity"`
	AllChecks   *bool   `json:"allChecks"`
	Checks      []idRef `json:"checks"`
	Groups      []idRef `json:"groups"`
	Apps        []idRef `json:"apps"`
	Contacts    []idRef `json:"contacts"`
}

func (a *Alert) CheckIDs() []int64   { return refIDs(a.Checks) }
func (a *Alert) GroupIDs() []int64   { return refIDs(a.Groups) }
func (a *Alert) AppIDs() []int64     { return refIDs(a.Apps) }
func (a *Alert) ContactIDs() []int64 { return refIDs(a.Contacts) }

// AlertInput is the create/update payload.
type AlertInput struct {
	Name        string
	MinSeverity string
	AllChecks   *bool
	CheckIDs    []int64
	GroupIDs    []int64
	AppIDs      []int64
	ContactIDs  []int64
}

func alertBody(input AlertInput) map[string]any {
	alert := map[string]any{"name": input.Name}
	if input.MinSeverity != "" {
		alert["minSeverity"] = input.MinSeverity
	}
	if input.AllChecks != nil {
		alert["allChecks"] = *input.AllChecks
	}
	if input.CheckIDs != nil {
		alert["checks"] = input.CheckIDs
	}
	if input.GroupIDs != nil {
		alert["groups"] = input.GroupIDs
	}
	if input.AppIDs != nil {
		alert["apps"] = input.AppIDs
	}
	if input.ContactIDs != nil {
		alert["contacts"] = idObjects(input.ContactIDs)
	}
	return map[string]any{"alert": alert}
}

func (c *Client) CreateAlert(ctx context.Context, input AlertInput) (*Alert, error) {
	return createObj[Alert](c, ctx, "/monitoring/alerts", "alert", alertBody(input))
}

func (c *Client) GetAlert(ctx context.Context, id int64) (*Alert, error) {
	return getByID[Alert](c, ctx, fmt.Sprintf("/monitoring/alerts/%d", id), "alert")
}

func (c *Client) GetAlertByName(ctx context.Context, name string) (*Alert, error) {
	return firstByName[Alert](c, ctx, "/monitoring/alerts", "alerts", name)
}

func (c *Client) UpdateAlert(ctx context.Context, id int64, input AlertInput) (*Alert, error) {
	return updateObj[Alert](c, ctx, fmt.Sprintf("/monitoring/alerts/%d", id), "alert", alertBody(input))
}

func (c *Client) DeleteAlert(ctx context.Context, id int64) error {
	return c.delete(ctx, fmt.Sprintf("/monitoring/alerts/%d", id), nil)
}

func (c *Client) ListAlerts(ctx context.Context) ([]Alert, error) {
	return listObjects[Alert](c, ctx, "/monitoring/alerts", "alerts")
}
