package web

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/14122mhl/safeops-go/internal/agent/rag"
	agentService "github.com/14122mhl/safeops-go/internal/agent/service"
	"github.com/14122mhl/safeops-go/internal/config"
	"github.com/14122mhl/safeops-go/internal/engine"
	"github.com/14122mhl/safeops-go/internal/trace"
)

type successRunner struct{}

func (successRunner) Run(_ context.Context, command []string) engine.Result {
	return engine.Result{ExitCode: 0, Command: command}
}

func testServer(t *testing.T) Server {
	t.Helper()
	root := t.TempDir()
	documents := filepath.Join(root, "docs")
	if err := os.MkdirAll(documents, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(documents, "release.md"), []byte("# Release\nUse dry-run before release."), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.RAG.Enabled = true
	cfg.RAG.Paths = []string{documents}
	store := trace.Store{Directory: filepath.Join(root, "runs")}
	return Server{Config: cfg, Agent: agentService.NewFromConfig(cfg, successRunner{}, store), TraceStore: store, Documents: rag.LocalSearcher{Paths: cfg.RAG.Paths}}
}

func TestStatusAndRAG(t *testing.T) {
	server := testServer(t).Handler()
	for _, path := range []string{"/api/status", "/api/rag"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()
		server.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("%s status = %d, body=%s", path, response.Code, response.Body.String())
		}
		if response.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("%s missing no-store header", path)
		}
	}
}

func TestGoalDefaultsToPlanOnly(t *testing.T) {
	serverValue := testServer(t)
	root := t.TempDir()
	playbook := filepath.Join(root, "demo.yml")
	inventory := filepath.Join(root, "inventory.ini")
	if err := os.WriteFile(playbook, []byte("- hosts: all\n  tasks:\n    - debug:\n        msg: safe\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inventory, []byte("localhost ansible_connection=local\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(GoalPayload{Goal: "preview development release", Playbook: playbook, Inventory: inventory, Environment: "dev"})
	request := httptest.NewRequest(http.MethodPost, "/api/goal", bytes.NewReader(body))
	response := httptest.NewRecorder()
	serverValue.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
	}
	var result struct {
		Response agentService.Response `json:"response"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Response.Status != "planned" || result.Response.ExitCode != 0 {
		t.Fatalf("response = %+v", result.Response)
	}
	if result.Response.Trace.Plan == nil || result.Response.Trace.Plan.Apply {
		t.Fatalf("plan must remain non-applying: %+v", result.Response.Trace.Plan)
	}
}

func TestGoalRejectsInvalidRequests(t *testing.T) {
	handler := testServer(t).Handler()
	tests := []struct {
		name, body string
		method     string
		want       int
	}{
		{name: "missing goal", body: `{}`, method: http.MethodPost, want: http.StatusBadRequest},
		{name: "unknown field", body: `{"goal":"x","unsafe":true}`, method: http.MethodPost, want: http.StatusBadRequest},
		{name: "wrong method", method: http.MethodGet, want: http.StatusMethodNotAllowed},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, "/api/goal", bytes.NewBufferString(test.body))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.want {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, test.want, response.Body.String())
			}
		})
	}
}

func TestRunsStartsEmpty(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/runs", nil)
	response := httptest.NewRecorder()
	testServer(t).Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"runs":[]`)) {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}
