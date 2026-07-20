package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/14122mhl/safeops-go/internal/model"
)

func TestGoalPlanOnlyEndToEnd(t *testing.T) {
	playbook, inventory := writeCLIPlaybookFixture(t)
	tracePath := filepath.Join(t.TempDir(), "goal-trace.json")
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--engine", "echo", "goal", "安全发布", "--playbook", playbook, "-i", inventory, "--env", "dev", "--plan-only", "--trace-out", tracePath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "Mode: dry-run") || !strings.Contains(stdout.String(), "Agent Plan-Only") {
		t.Fatalf("stdout=%s", stdout.String())
	}
	data, err := os.ReadFile(tracePath)
	if err != nil {
		t.Fatal(err)
	}
	var value model.RunTrace
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatalf("trace JSON: %v", err)
	}
	if value.Status != "planned" || value.Plan.Apply {
		t.Fatalf("trace = %+v", value)
	}
}

func TestGoalWritesClarificationTrace(t *testing.T) {
	tracePath := filepath.Join(t.TempDir(), "clarify.json")
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--engine", "echo", "goal", "发布到 dev", "--trace-out", tracePath}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "Agent Clarify") {
		t.Fatalf("stdout=%s", stdout.String())
	}
	if _, err := os.Stat(tracePath); err != nil {
		t.Fatal(err)
	}
}
