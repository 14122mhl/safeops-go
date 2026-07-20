package check

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/14122mhl/safeops-go/internal/config"
	"github.com/14122mhl/safeops-go/internal/engine"
)

type fakeRunner struct {
	results []engine.Result
	calls   [][]string
}

func (runner *fakeRunner) Run(_ context.Context, command []string) engine.Result {
	runner.calls = append(runner.calls, append([]string(nil), command...))
	result := engine.Result{ExitCode: 0, Command: command}
	if len(runner.results) > 0 {
		result = runner.results[0]
		runner.results = runner.results[1:]
	}
	return result
}

func TestPreflightRunsStaticAndAnsibleChecks(t *testing.T) {
	directory := t.TempDir()
	playbook := filepath.Join(directory, "demo.yml")
	inventory := filepath.Join(directory, "inventory.ini")
	if err := os.WriteFile(playbook, []byte("- hosts: all\n  tasks:\n    - ansible.builtin.debug:\n        msg: ok\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inventory, []byte("[all]\nlocalhost\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{}
	results := Preflight(context.Background(), Request{Playbook: playbook, Inventory: inventory, Environment: "dev", Engine: "echo"}, config.Default().Risk, runner)
	if HasFailures(results) {
		t.Fatalf("results contain failures: %+v", results)
	}
	if len(runner.calls) != 3 {
		t.Fatalf("runner calls = %d, want 3", len(runner.calls))
	}
}

func TestPreflightStopsBeforeRunnerOnMissingPlaybook(t *testing.T) {
	runner := &fakeRunner{}
	results := Preflight(context.Background(), Request{Playbook: "missing.yml", Environment: "dev"}, config.Default().Risk, runner)
	if !HasFailures(results) {
		t.Fatalf("results = %+v, want failure", results)
	}
	if len(runner.calls) != 0 {
		t.Fatalf("runner calls = %d, want 0", len(runner.calls))
	}
}

func TestPreflightReportsSyntaxFailure(t *testing.T) {
	directory := t.TempDir()
	playbook := filepath.Join(directory, "demo.yml")
	if err := os.WriteFile(playbook, []byte("- hosts: all\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{results: []engine.Result{{ExitCode: 2, Stderr: "syntax failed"}}}
	results := Preflight(context.Background(), Request{Playbook: playbook, Environment: "dev"}, config.Default().Risk, runner)
	if !HasFailures(results) {
		t.Fatalf("results = %+v, want failure", results)
	}
}
