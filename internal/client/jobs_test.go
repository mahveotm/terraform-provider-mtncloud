package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func jobFromInput(t *testing.T, in JobInput) map[string]any {
	t.Helper()
	var job map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/jobs" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		job = payload["job"].(map[string]any)
		_ = json.NewEncoder(w).Encode(map[string]any{"job": map[string]any{"id": 3, "name": job["name"]}})
	}))
	defer server.Close()

	c, err := New(Config{URL: server.URL, Token: "test-token", Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.CreateJob(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	return job
}

func TestJobScheduleModeManual(t *testing.T) {
	t.Parallel()
	wf := int64(11)
	job := jobFromInput(t, JobInput{Name: "j", WorkflowID: &wf, ScheduleMode: "manual"})
	if job["scheduleMode"] != "manual" {
		t.Fatalf("expected scheduleMode=manual, got %#v", job["scheduleMode"])
	}
	if job["workflow"].(map[string]any)["id"].(float64) != 11 {
		t.Fatalf("unexpected workflow ref: %#v", job["workflow"])
	}
}

func TestJobScheduleModeDateTime(t *testing.T) {
	t.Parallel()
	task := int64(9)
	job := jobFromInput(t, JobInput{Name: "j", TaskID: &task, ScheduleMode: "date_time", DateTime: "2026-07-01T02:00:00Z"})
	if job["scheduleMode"] != "dateTime" {
		t.Fatalf("expected scheduleMode=dateTime, got %#v", job["scheduleMode"])
	}
	if job["dateTime"] != "2026-07-01T02:00:00Z" {
		t.Fatalf("expected dateTime echoed, got %#v", job["dateTime"])
	}
}

func TestJobScheduleModeSchedule(t *testing.T) {
	t.Parallel()
	wf := int64(11)
	sched := int64(54)
	job := jobFromInput(t, JobInput{Name: "j", WorkflowID: &wf, ScheduleMode: "schedule", ExecuteScheduleID: &sched})
	if job["scheduleMode"] != "54" {
		t.Fatalf("expected scheduleMode=54 (schedule id), got %#v", job["scheduleMode"])
	}
}

func TestJobInstanceTargets(t *testing.T) {
	t.Parallel()
	wf := int64(11)
	job := jobFromInput(t, JobInput{Name: "j", WorkflowID: &wf, ScheduleMode: "manual", TargetType: "instance", Targets: []int64{123, 456}})
	if job["targetType"] != "instance" {
		t.Fatalf("expected targetType=instance, got %#v", job["targetType"])
	}
	targets := job["targets"].([]any)
	if len(targets) != 2 || targets[0].(map[string]any)["refId"].(float64) != 123 {
		t.Fatalf("unexpected targets: %#v", targets)
	}
}
