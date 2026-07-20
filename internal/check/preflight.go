package check

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/14122mhl/safeops-go/internal/analysis"
	"github.com/14122mhl/safeops-go/internal/config"
	"github.com/14122mhl/safeops-go/internal/engine"
	"github.com/14122mhl/safeops-go/internal/model"
)

var allowedEnvironments = map[string]bool{"dev": true, "test": true, "stage": true, "staging": true, "prod": true, "production": true}

// Request contains all inputs needed for preflight validation.
type Request struct {
	Playbook, Inventory, Limit, Environment, Engine string
	ExtraVars                                       []string
}

// Preflight performs local, static, and Ansible validation.
func Preflight(ctx context.Context, request Request, rules map[string]config.RiskRule, runner engine.Runner) []model.CheckResult {
	results := localChecks(request)
	if HasFailures(results) {
		return results
	}
	playbookAnalysis, err := analysis.Analyze(request.Playbook, rules)
	if err != nil {
		return append(results, model.CheckResult{Name: "static_analysis", Status: model.CheckFail, Message: err.Error(), Remediation: "fix playbook YAML before continuing"})
	}
	status := model.CheckPass
	message := fmt.Sprintf("%d task(s), overall risk=%s", len(playbookAnalysis.Tasks), playbookAnalysis.OverallRisk)
	if playbookAnalysis.OverallRisk == model.RiskHigh {
		status = model.CheckWarn
	}
	results = append(results, model.CheckResult{Name: "static_analysis", Status: status, Message: message})
	results = append(results, ansibleChecks(ctx, request, runner)...)
	return results
}

func localChecks(request Request) []model.CheckResult {
	results := []model.CheckResult{}
	if info, err := os.Stat(request.Playbook); err != nil || info.IsDir() {
		results = append(results, model.CheckResult{Name: "playbook", Status: model.CheckFail, Message: "playbook file not found", Remediation: "provide an existing playbook path"})
	} else {
		results = append(results, model.CheckResult{Name: "playbook", Status: model.CheckPass, Message: request.Playbook})
	}
	if request.Inventory == "" {
		results = append(results, model.CheckResult{Name: "inventory", Status: model.CheckWarn, Message: "inventory not provided"})
	} else if info, err := os.Stat(request.Inventory); err != nil || info.IsDir() {
		results = append(results, model.CheckResult{Name: "inventory", Status: model.CheckFail, Message: "inventory path not found", Remediation: "provide an existing inventory path"})
	} else {
		results = append(results, model.CheckResult{Name: "inventory", Status: model.CheckPass, Message: request.Inventory})
	}
	if !allowedEnvironments[strings.ToLower(request.Environment)] {
		results = append(results, model.CheckResult{Name: "environment", Status: model.CheckFail, Message: fmt.Sprintf("unsupported environment %q", request.Environment), Remediation: "use dev, test, stage, staging, prod, or production"})
	} else {
		results = append(results, model.CheckResult{Name: "environment", Status: model.CheckPass, Message: request.Environment})
	}
	return results
}

func ansibleChecks(ctx context.Context, request Request, runner engine.Runner) []model.CheckResult {
	modes := []struct {
		name    string
		mode    engine.Mode
		failure model.CheckStatus
	}{
		{"ansible_syntax", engine.ModeSyntax, model.CheckFail},
		{"ansible_list_hosts", engine.ModeListHosts, model.CheckFail},
		{"ansible_list_tasks", engine.ModeListTasks, model.CheckWarn},
	}
	results := make([]model.CheckResult, 0, len(modes))
	for _, validation := range modes {
		if request.Inventory == "" && validation.mode != engine.ModeSyntax {
			continue
		}
		command, _ := engine.BuildCommand(engine.CommandRequest{Engine: request.Engine, Playbook: request.Playbook, Inventory: request.Inventory, Limit: request.Limit, Environment: request.Environment, ExtraVars: request.ExtraVars, Mode: validation.mode})
		result := runner.Run(ctx, command)
		if result.ExitCode == 0 {
			results = append(results, model.CheckResult{Name: validation.name, Status: model.CheckPass, Message: "validation succeeded"})
		} else if result.ExitCode == 127 {
			results = append(results, model.CheckResult{Name: validation.name, Status: model.CheckWarn, Message: "ansible-playbook not found; validation skipped", Remediation: "install Ansible or select an existing executor"})
		} else {
			message := strings.TrimSpace(result.Stderr)
			if message == "" {
				message = strings.TrimSpace(result.Stdout)
			}
			results = append(results, model.CheckResult{Name: validation.name, Status: validation.failure, Message: message})
		}
	}
	return results
}

// HasFailures reports whether any check blocks execution.
func HasFailures(results []model.CheckResult) bool {
	for _, result := range results {
		if result.Status == model.CheckFail {
			return true
		}
	}
	return false
}
