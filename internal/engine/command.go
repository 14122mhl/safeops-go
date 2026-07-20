// Package engine builds and executes external automation commands.
package engine

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
)

// Mode identifies a non-mutating preview or an explicitly authorized apply.
type Mode string

const (
	ModeDryRun    Mode = "dry-run"
	ModeApply     Mode = "apply"
	ModeSyntax    Mode = "syntax"
	ModeListHosts Mode = "list-hosts"
	ModeListTasks Mode = "list-tasks"
)

// CommandRequest contains normalized Ansible arguments.
type CommandRequest struct {
	Engine      string
	Playbook    string
	Inventory   string
	Limit       string
	Environment string
	ExtraVars   []string
	Mode        Mode
}

// BuildCommand returns an argv slice without invoking a shell.
func BuildCommand(request CommandRequest) ([]string, error) {
	if request.Playbook == "" {
		return nil, errors.New("playbook is required")
	}
	engine := request.Engine
	if engine == "" || engine == "ansible" {
		engine = "ansible-playbook"
	} else if path, err := exec.LookPath(engine); err == nil {
		engine = path
	}
	args := []string{engine}
	switch request.Mode {
	case ModeDryRun:
		args = append(args, "--check", "--diff")
	case ModeApply:
	case ModeSyntax:
		args = append(args, "--syntax-check")
	case ModeListHosts:
		args = append(args, "--list-hosts")
	case ModeListTasks:
		args = append(args, "--list-tasks")
	default:
		return nil, errors.New("unknown execution mode")
	}
	if request.Inventory != "" {
		args = append(args, "-i", request.Inventory)
	}
	if request.Limit != "" {
		args = append(args, "--limit", request.Limit)
	}
	if request.Environment != "" {
		args = append(args, "-e", "env="+request.Environment)
	}
	for _, value := range request.ExtraVars {
		args = append(args, "-e", value)
	}
	return append(args, request.Playbook), nil
}

// Result captures an external command without losing its argv evidence.
type Result struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Command  []string
}

// Runner executes an argv slice with cancellation and no shell expansion.
type Runner interface {
	Run(context.Context, []string) Result
}

// ExecRunner uses os/exec and is the production Runner implementation.
type ExecRunner struct{}

// Run executes command until completion or context cancellation.
func (ExecRunner) Run(ctx context.Context, command []string) Result {
	if len(command) == 0 {
		return Result{ExitCode: 2, Stderr: "empty command"}
	}
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		var exitError *exec.ExitError
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			exitCode = 124
		} else if errors.Is(ctx.Err(), context.Canceled) {
			exitCode = 130
		} else if errors.As(err, &exitError) {
			exitCode = exitError.ExitCode()
		} else if errors.Is(err, exec.ErrNotFound) {
			exitCode = 127
		} else {
			exitCode = 1
		}
		if stderr.Len() == 0 {
			stderr.WriteString(err.Error())
		}
	}
	return Result{ExitCode: exitCode, Stdout: stdout.String(), Stderr: stderr.String(), Command: append([]string(nil), command...)}
}
