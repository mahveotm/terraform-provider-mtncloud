package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCreateWorkflowPayload(t *testing.T) {
	t.Parallel()
	var taskSet map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/task-sets" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		taskSet = payload["taskSet"].(map[string]any)
		_ = json.NewEncoder(w).Encode(map[string]any{"taskSet": map[string]any{"id": 11, "name": taskSet["name"]}})
	}))
	defer server.Close()

	c, err := New(Config{URL: server.URL, Token: "test-token", Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.CreateWorkflow(context.Background(), WorkflowInput{
		Name: "maint",
		Type: "operation",
		Tasks: []WorkflowTask{
			{TaskID: 7, Phase: "operation"},
			{TaskID: 9, Phase: "operation"},
		},
	}); err != nil {
		t.Fatal(err)
	}

	if taskSet["type"] != "operation" {
		t.Fatalf("expected type=operation, got %#v", taskSet["type"])
	}
	tasks := taskSet["tasks"].([]any)
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %#v", tasks)
	}
	first := tasks[0].(map[string]any)
	if first["taskId"].(float64) != 7 || first["taskPhase"] != "operation" {
		t.Fatalf("unexpected first task: %#v", first)
	}
}
