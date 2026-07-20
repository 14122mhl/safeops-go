package deepseek

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/14122mhl/safeops-go/internal/provider"
)

func TestParseGoalUsesCompatibleProtocolAndBoundsPaths(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("authorization = %q", request.Header.Get("Authorization"))
		}
		var payload map[string]any
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		messages, ok := payload["messages"].([]any)
		if !ok || len(messages) != 2 {
			t.Fatalf("messages = %#v", payload["messages"])
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"playbook\":\"invented.yml\",\"inventory\":\"inventory.ini\",\"environment\":\"stage\",\"apply_intent\":true,\"confidence\":1.4}"}}]}`))
	}))
	defer server.Close()
	client := Client{APIKey: "secret", BaseURL: server.URL, Model: "test", HTTPClient: server.Client()}
	hints, err := client.ParseGoal(context.Background(), provider.Request{Goal: "deploy", PlaybookCandidates: []string{"demo.yml"}, InventoryCandidates: []string{"inventory.ini"}})
	if err != nil {
		t.Fatalf("ParseGoal() error = %v", err)
	}
	if hints.Playbook != "" {
		t.Fatalf("invented playbook was accepted: %q", hints.Playbook)
	}
	if hints.Inventory != "inventory.ini" || hints.Environment != "stage" {
		t.Fatalf("hints = %+v", hints)
	}
	if !hints.ApplyIntent {
		t.Fatal("ApplyIntent = false, want semantic intent")
	}
	if hints.Confidence != 1 {
		t.Fatalf("Confidence = %v, want 1", hints.Confidence)
	}
}

func TestParseGoalRejectsMissingKey(t *testing.T) {
	if _, err := (Client{}).ParseGoal(context.Background(), provider.Request{}); err == nil {
		t.Fatal("ParseGoal() error = nil")
	}
}
