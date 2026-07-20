package cli

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), nil, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "safeops: safe change agent platform") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunConfigInitAndShow(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	var stdout, stderr bytes.Buffer
	if code := Run(context.Background(), []string{"--config", path, "config", "init"}, &stdout, &stderr); code != 0 {
		t.Fatalf("init code = %d, stderr = %s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Run(context.Background(), []string{"--config", path, "config", "show"}, &stdout, &stderr); code != 0 {
		t.Fatalf("show code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "default_engine: ansible") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunUnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"unknown"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("Run() code = %d, want 2", code)
	}
}
