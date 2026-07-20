// Package engine builds and executes external automation commands.
package engine

import (
	"errors"
	"os/exec"
)

// Mode identifies a non-mutating preview or an explicitly authorized apply.
type Mode string

const (
	ModeDryRun Mode = "dry-run"
	ModeApply  Mode = "apply"
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
