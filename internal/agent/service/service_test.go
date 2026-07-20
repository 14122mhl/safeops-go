package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/14122mhl/safeops-go/internal/config"
	"github.com/14122mhl/safeops-go/internal/engine"
	"github.com/14122mhl/safeops-go/internal/trace"
)

type fakeRunner struct {
	results []engine.Result
	calls   [][]string
}

func (runner *fakeRunner) Run(_ context.Context, command []string) engine.Result {
	runner.calls = append(runner.calls, append([]string(nil), command...))
	if len(runner.results) == 0 {
		return engine.Result{Command: command}
	}
	result := runner.results[0]
	runner.results = runner.results[1:]
	result.Command = command
	return result
}

type captureSink struct {
	lines          []string
	stdout, stderr strings.Builder
}

func (sink *captureSink) Line(value string)   { sink.lines = append(sink.lines, value) }
func (sink *captureSink) Stdout(value string) { sink.stdout.WriteString(value) }
func (sink *captureSink) Stderr(value string) { sink.stderr.WriteString(value) }

func serviceFixture(t *testing.T) (Service, string, string, *fakeRunner) {
	t.Helper()
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
	clock := func() time.Time { return time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC) }
	return Service{Config: config.Default(), Runner: runner, TraceStore: trace.Store{Directory: filepath.Join(directory, "runs")}, Now: clock}, playbook, inventory, runner
}

func TestRunPlanOnlyWritesAuditableTrace(t *testing.T) {
	service, playbook, inventory, runner := serviceFixture(t)
	response := service.Run(context.Background(), Request{Goal: "安全发布", Playbook: playbook, Inventory: inventory, Environment: "dev", PlanOnly: true}, &captureSink{})
	if response.ExitCode != 0 || response.Status != "planned" {
		t.Fatalf("response = %+v", response)
	}
	if len(runner.calls) != 3 {
		t.Fatalf("runner calls = %d, want validation only", len(runner.calls))
	}
	if response.Trace.Plan.TemplateID != "release" {
		t.Fatalf("template = %q", response.Trace.Plan.TemplateID)
	}
	if _, err := os.Stat(response.TracePath); err != nil {
		t.Fatalf("trace missing: %v", err)
	}
}

func TestRunExecutesDryRunAndWritesLog(t *testing.T) {
	service, playbook, inventory, runner := serviceFixture(t)
	runner.results = []engine.Result{{}, {}, {}, {Stdout: "ok\n"}}
	response := service.Run(context.Background(), Request{Goal: "发布", Playbook: playbook, Inventory: inventory, Environment: "dev"}, &captureSink{})
	if response.ExitCode != 0 || response.Status != "success" {
		t.Fatalf("response = %+v", response)
	}
	if len(runner.calls) != 4 {
		t.Fatalf("runner calls = %d, want 4", len(runner.calls))
	}
	if len(runner.calls[3]) < 3 || runner.calls[3][1] != "--check" {
		t.Fatalf("execution command = %v", runner.calls[3])
	}
	if _, err := os.Stat(response.LogPath); err != nil {
		t.Fatalf("log missing: %v", err)
	}
}

func TestRunNeedsClarificationWithoutPlaybook(t *testing.T) {
	service, _, _, runner := serviceFixture(t)
	response := service.Run(context.Background(), Request{Goal: "发布到 dev"}, &captureSink{})
	if response.Status != "needs_clarification" || response.ExitCode != 1 {
		t.Fatalf("response = %+v", response)
	}
	if len(runner.calls) != 0 {
		t.Fatalf("runner calls = %d, want 0", len(runner.calls))
	}
	if response.TracePath == "" {
		t.Fatal("clarification trace was not written")
	}
}

func TestRunBlocksApplyWithoutApproval(t *testing.T) {
	service, playbook, inventory, runner := serviceFixture(t)
	response := service.Run(context.Background(), Request{Goal: "发布", Playbook: playbook, Inventory: inventory, Environment: "dev", ExplicitApply: true}, &captureSink{})
	if response.Status != "failed" || response.Error != "approval gate failed" {
		t.Fatalf("response = %+v", response)
	}
	if len(runner.calls) != 3 {
		t.Fatalf("runner calls = %d, want no execution", len(runner.calls))
	}
}
