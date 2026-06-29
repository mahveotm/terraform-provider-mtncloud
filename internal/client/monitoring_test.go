package client

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestContactBodyOmitsEmpty(t *testing.T) {
	t.Parallel()

	body := contactBody(ContactInput{Name: "On-Call", EmailAddress: "oncall@mtn.ng"}, false)
	contact := body["contact"].(map[string]any)
	if contact["name"] != "On-Call" || contact["emailAddress"] != "oncall@mtn.ng" {
		t.Fatalf("unexpected contact: %#v", contact)
	}
	if _, ok := contact["smsAddress"]; ok {
		t.Fatalf("expected empty smsAddress to be omitted: %#v", contact)
	}
	if _, ok := contact["slackHook"]; ok {
		t.Fatalf("expected empty slackHook to be omitted: %#v", contact)
	}
}

func TestContactBodySendsEmptyOnUpdate(t *testing.T) {
	t.Parallel()

	body := contactBody(ContactInput{Name: "On-Call", EmailAddress: "oncall@mtn.ng"}, true)
	contact := body["contact"].(map[string]any)
	if contact["smsAddress"] != "" || contact["slackHook"] != "" {
		t.Fatalf("expected empty update fields to be sent for clearing, got %#v", contact)
	}
}

func TestCheckBodyWrapsCheckTypeAndConfig(t *testing.T) {
	t.Parallel()

	interval := int64(120000)
	up := true
	body := checkBody(CheckInput{
		Name:          "web-home",
		CheckType:     "webGetCheck",
		CheckInterval: &interval,
		InUptime:      &up,
		Severity:      "critical",
		Config:        map[string]any{"webUrl": "https://mtn.ng"},
	})
	check := body["check"].(map[string]any)
	if check["name"] != "web-home" || check["severity"] != "critical" {
		t.Fatalf("unexpected check: %#v", check)
	}
	ct := check["checkType"].(map[string]any)
	if ct["code"] != "webGetCheck" {
		t.Fatalf("expected checkType.code=webGetCheck, got %#v", ct)
	}
	if check["checkInterval"] != interval || check["inUptime"] != true {
		t.Fatalf("unexpected interval/uptime: %#v", check)
	}
	cfg := check["config"].(map[string]any)
	if cfg["webUrl"] != "https://mtn.ng" {
		t.Fatalf("unexpected config: %#v", cfg)
	}
}

func TestCheckBodyOmitsConfigWhenEmpty(t *testing.T) {
	t.Parallel()

	body := checkBody(CheckInput{Name: "ping", CheckType: "pingCheck"})
	check := body["check"].(map[string]any)
	if _, ok := check["config"]; ok {
		t.Fatalf("expected empty config to be omitted: %#v", check)
	}
}

func TestMonitoringGroupBodyUsesCheckGroupWrapper(t *testing.T) {
	t.Parallel()

	minHappy := int64(1)
	body := monitoringGroupBody(MonitoringGroupInput{Name: "frontends", MinHappy: &minHappy, CheckIDs: []int64{3, 7}})
	group, ok := body["checkGroup"].(map[string]any)
	if !ok {
		t.Fatalf("expected checkGroup wrapper, got %#v", body)
	}
	if group["name"] != "frontends" || group["minHappy"] != minHappy {
		t.Fatalf("unexpected group: %#v", group)
	}
	checks := group["checks"].([]int64)
	if len(checks) != 2 || checks[1] != 7 {
		t.Fatalf("unexpected checks: %#v", checks)
	}
}

func TestAlertBodyWrapsContactsAsObjects(t *testing.T) {
	t.Parallel()

	all := false
	body := alertBody(AlertInput{Name: "page-oncall", MinSeverity: "critical", AllChecks: &all, CheckIDs: []int64{5}, ContactIDs: []int64{11}})
	alert := body["alert"].(map[string]any)
	if alert["name"] != "page-oncall" || alert["minSeverity"] != "critical" || alert["allChecks"] != false {
		t.Fatalf("unexpected alert: %#v", alert)
	}
	checks := alert["checks"].([]int64)
	if len(checks) != 1 || checks[0] != 5 {
		t.Fatalf("unexpected checks: %#v", checks)
	}
	contacts := alert["contacts"].([]map[string]any)
	if len(contacts) != 1 || contacts[0]["id"] != int64(11) {
		t.Fatalf("expected contacts as [{id}], got %#v", contacts)
	}
}

func TestCreateAlertRoundTrip(t *testing.T) {
	t.Parallel()

	c, closeFn := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/monitoring/alerts" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		alert := payload["alert"].(map[string]any)
		if alert["name"] != "a1" {
			t.Fatalf("unexpected alert payload: %#v", alert)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"alert": map[string]any{
				"id":       42,
				"name":     "a1",
				"contacts": []map[string]any{{"id": 11}},
			},
		})
	})
	defer closeFn()

	alert, err := c.CreateAlert(context.Background(), AlertInput{Name: "a1", ContactIDs: []int64{11}})
	if err != nil {
		t.Fatal(err)
	}
	if alert.ID != 42 || len(alert.ContactIDs()) != 1 || alert.ContactIDs()[0] != 11 {
		t.Fatalf("unexpected alert: %#v", alert)
	}
}
