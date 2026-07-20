package engine

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestExecRunnerCapturesOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses POSIX printf")
	}
	result := (ExecRunner{}).Run(context.Background(), []string{"printf", "safeops"})
	if result.ExitCode != 0 || result.Stdout != "safeops" {
		t.Fatalf("result = %+v", result)
	}
}

func TestExecRunnerHonorsTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses POSIX sleep")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	result := (ExecRunner{}).Run(ctx, []string{"sleep", "1"})
	if result.ExitCode != 124 {
		t.Fatalf("ExitCode = %d, want 124; stderr=%s", result.ExitCode, result.Stderr)
	}
	if !strings.Contains(result.Stderr, "killed") && !strings.Contains(result.Stderr, "deadline") {
		t.Fatalf("Stderr = %q, want cancellation evidence", result.Stderr)
	}
}
