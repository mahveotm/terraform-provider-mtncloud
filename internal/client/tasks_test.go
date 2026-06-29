package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// taskServer captures the POST /api/tasks payload and returns a minimal task.
func taskServer(t *testing.T, capture *map[string]any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tasks" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		*capture = payload["task"].(map[string]any)
		_ = json.NewEncoder(w).Encode(map[string]any{"task": map[string]any{"id": 7, "name": (*capture)["name"]}})
	}))
}

func newTaskClient(t *testing.T, url string) *Client {
	t.Helper()
	c, err := New(Config{URL: url, Token: "test-token", Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestCreateShellTaskPayload(t *testing.T) {
	t.Parallel()
	var task map[string]any
	server := taskServer(t, &task)
	defer server.Close()

	if _, err := newTaskClient(t, server.URL).CreateTask(context.Background(), TaskInput{
		Type:          "shell",
		Name:          "deploy",
		SourceType:    "local",
		Content:       "echo hi",
		ExecuteTarget: "remote",
		Sudo:          true,
		Host:          "10.0.0.5",
		Password:      "secret",
	}); err != nil {
		t.Fatal(err)
	}

	if task["taskType"].(map[string]any)["code"] != "script" {
		t.Fatalf("expected taskType.code=script, got %#v", task["taskType"])
	}
	file := task["file"].(map[string]any)
	if file["sourceType"] != "local" || file["content"] != "echo hi" {
		t.Fatalf("unexpected file: %#v", file)
	}
	opts := task["taskOptions"].(map[string]any)
	if opts["shell.sudo"] != "on" {
		t.Fatalf("expected shell.sudo=on, got %#v", opts)
	}
	if opts["host"] != "10.0.0.5" || opts["password"] != "secret" {
		t.Fatalf("expected remote host/password, got %#v", opts)
	}
}

func TestCreateAnsibleTaskPayload(t *testing.T) {
	t.Parallel()
	var task map[string]any
	server := taskServer(t, &task)
	defer server.Close()

	if _, err := newTaskClient(t, server.URL).CreateTask(context.Background(), TaskInput{
		Type:     "ansible",
		Name:     "site",
		Playbook: "site.yml",
		Tags:     "web",
	}); err != nil {
		t.Fatal(err)
	}

	if task["taskType"].(map[string]any)["code"] != "ansibleTask" {
		t.Fatalf("expected taskType.code=ansibleTask, got %#v", task["taskType"])
	}
	if _, ok := task["file"]; ok {
		t.Fatalf("ansible task should not send a file block: %#v", task["file"])
	}
	opts := task["taskOptions"].(map[string]any)
	if opts["ansiblePlaybook"] != "site.yml" || opts["ansibleTags"] != "web" {
		t.Fatalf("unexpected ansible options: %#v", opts)
	}
}

func TestCreateEmailTaskPayload(t *testing.T) {
	t.Parallel()
	var task map[string]any
	server := taskServer(t, &task)
	defer server.Close()

	if _, err := newTaskClient(t, server.URL).CreateTask(context.Background(), TaskInput{
		Type:         "email",
		Name:         "notify",
		SourceType:   "local",
		Content:      "<p>hi</p>",
		EmailAddress: "ops@example.com",
		Subject:      "Done",
	}); err != nil {
		t.Fatal(err)
	}

	if task["taskType"].(map[string]any)["code"] != "email" {
		t.Fatalf("expected taskType.code=email, got %#v", task["taskType"])
	}
	opts := task["taskOptions"].(map[string]any)
	if opts["emailAddress"] != "ops@example.com" || opts["emailSubject"] != "Done" {
		t.Fatalf("unexpected email options: %#v", opts)
	}
}

func TestTaskTypeRoundTrip(t *testing.T) {
	t.Parallel()
	for _, friendly := range TaskTypes {
		code := taskTypeCodes[friendly]
		if got := TaskTypeFromCode(code); got != friendly {
			t.Fatalf("round-trip failed: %q -> %q -> %q", friendly, code, got)
		}
	}
}
