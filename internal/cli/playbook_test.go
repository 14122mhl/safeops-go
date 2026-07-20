package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeCLIPlaybookFixture(t *testing.T) (string, string) {
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
	return playbook, inventory
}

func TestInspectCommand(t *testing.T) {
	playbook, _ := writeCLIPlaybookFixture(t)
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"inspect", playbook}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Overall Risk: LOW") {
		t.Fatalf("stdout=%s", stdout.String())
	}
}

func TestCheckCommandWithEchoExecutor(t *testing.T) {
	playbook, inventory := writeCLIPlaybookFixture(t)
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--engine", "echo", "check", playbook, "-i", inventory, "--env", "dev"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), "[PASS] ansible_syntax") {
		t.Fatalf("stdout=%s", stdout.String())
	}
}

func TestRunCommandDefaultsToDryRun(t *testing.T) {
	playbook, inventory := writeCLIPlaybookFixture(t)
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--engine", "echo", "run", playbook, "-i", inventory, "--env", "dev"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), "Execution mode: dry-run") || !strings.Contains(stdout.String(), "--check --diff") {
		t.Fatalf("stdout=%s", stdout.String())
	}
}

func TestRunCommandBlocksApplyWithoutApproval(t *testing.T) {
	playbook, inventory := writeCLIPlaybookFixture(t)
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--engine", "echo", "run", playbook, "-i", inventory, "--env", "dev", "--apply"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code=%d, want 1; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "apply requires explicit operator approval") {
		t.Fatalf("stdout=%s", stdout.String())
	}
}
