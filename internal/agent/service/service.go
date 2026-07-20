// Package service coordinates planning, analysis, checks, policy, execution, and tracing.
package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/14122mhl/safeops-go/internal/agent/planner"
	"github.com/14122mhl/safeops-go/internal/agent/policy"
	changeTemplate "github.com/14122mhl/safeops-go/internal/agent/template"
	"github.com/14122mhl/safeops-go/internal/analysis"
	"github.com/14122mhl/safeops-go/internal/check"
	"github.com/14122mhl/safeops-go/internal/config"
	"github.com/14122mhl/safeops-go/internal/engine"
	"github.com/14122mhl/safeops-go/internal/model"
	"github.com/14122mhl/safeops-go/internal/trace"
)

// Sink receives human-readable progress and process output.
type Sink interface {
	Line(string)
	Stdout(string)
	Stderr(string)
}

type discardSink struct{}

func (discardSink) Line(string)   {}
func (discardSink) Stdout(string) {}
func (discardSink) Stderr(string) {}

// Request separates semantic goal text from trusted operator controls.
type Request struct {
	Goal, Playbook, Inventory, Environment, Limit string
	ExtraVars                                     []string
	ExplicitApply, Approved, PlanOnly             bool
	ProductionConfirm, TraceOut, Engine           string
	Timeout                                       time.Duration
}

// Response is suitable for CLI, HTTP, and future UI adapters.
type Response struct {
	ExitCode                                 int
	Status, RunID, TracePath, LogPath, Error string
	Trace                                    model.RunTrace
}

// Service owns dependencies used by one or more goal requests.
type Service struct {
	Config     config.Config
	Runner     engine.Runner
	TraceStore trace.Store
	Now        func() time.Time
}

// Run evaluates and optionally executes a natural-language change goal.
func (service Service) Run(ctx context.Context, request Request, output Sink) Response {
	if output == nil {
		output = discardSink{}
	}
	now := service.Now
	if now == nil {
		now = time.Now
	}
	runner := service.Runner
	if runner == nil {
		runner = engine.ExecRunner{}
	}
	started := now().UTC()
	runTrace := model.RunTrace{RunID: trace.NewRunID(started), StartedAt: started, Goal: request.Goal, Status: "running", Steps: []model.TraceStep{}}

	plan := planner.Build(planner.Request{Goal: request.Goal, Playbook: request.Playbook, Inventory: request.Inventory, Environment: request.Environment, Limit: request.Limit, ExtraVars: request.ExtraVars, ExplicitApply: request.ExplicitApply, DefaultEnv: service.Config.Settings.DefaultEnv})
	if matched, ok := changeTemplate.Match(request.Goal); ok {
		plan.TemplateID = matched.ID
		plan.RecommendedSteps = append([]string(nil), matched.RecommendedSteps...)
		plan.RiskNotes = append([]string(nil), matched.RiskNotes...)
		plan.Notes = append(plan.Notes, "template matched: "+matched.ID)
	}
	runTrace.Plan = &plan
	emitPlan(output, plan)
	if len(plan.MissingFields) > 0 {
		runTrace.Status = "needs_clarification"
		runTrace.Error = "missing required fields: " + strings.Join(plan.MissingFields, ", ")
		output.Line("Agent Clarify")
		output.Line(runTrace.Error)
		return service.finish(runTrace, request.TraceOut, 1, "", output, now)
	}

	playbookAnalysis, err := analysis.Analyze(plan.Playbook, service.Config.Risk)
	if err != nil {
		runTrace.Status = "failed"
		runTrace.Error = err.Error()
		output.Line("[FAIL] analysis: " + err.Error())
		return service.finish(runTrace, request.TraceOut, 1, "", output, now)
	}
	runTrace.Analysis = &playbookAnalysis
	output.Line(fmt.Sprintf("Agent Analyze: risk=%s tasks=%d", playbookAnalysis.OverallRisk, len(playbookAnalysis.Tasks)))

	timeout := request.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	checkContext, cancelChecks := context.WithTimeout(ctx, timeout)
	checks := check.Preflight(checkContext, check.Request{Playbook: plan.Playbook, Inventory: plan.Inventory, Limit: plan.Limit, Environment: plan.Environment, ExtraVars: plan.ExtraVars, Engine: request.Engine}, service.Config.Risk, runner)
	cancelChecks()
	runTrace.Steps = append(runTrace.Steps, model.TraceStep{Name: "check", Checks: checks})
	for _, result := range checks {
		output.Line(fmt.Sprintf("[%s] %s: %s", result.Status, result.Name, result.Message))
	}
	if check.HasFailures(checks) {
		runTrace.Status = "failed"
		runTrace.Error = "preflight checks failed"
		return service.finish(runTrace, request.TraceOut, 1, "", output, now)
	}

	decision := policy.Evaluate(policy.Request{ExplicitApply: plan.Apply, Approved: request.Approved, Environment: plan.Environment, ProductionConfirm: request.ProductionConfirm, RequireProdConfirm: service.Config.Settings.RequireProdConfirm, PlanConfidence: plan.Confidence, MinimumConfidence: service.Config.Settings.MinGoalConfidenceToApply, ChecksPassed: true})
	runTrace.Steps = append(runTrace.Steps, model.TraceStep{Name: "approval", Allowed: boolPointer(decision.Allowed), Reasons: decision.Reasons})
	for _, reason := range decision.Reasons {
		output.Line("[POLICY] " + reason)
	}
	if !decision.Allowed {
		runTrace.Status = "failed"
		runTrace.Error = "approval gate failed"
		return service.finish(runTrace, request.TraceOut, 1, "", output, now)
	}

	mode := engine.ModeDryRun
	if plan.Apply {
		mode = engine.ModeApply
	}
	command, err := engine.BuildCommand(engine.CommandRequest{Engine: request.Engine, Playbook: plan.Playbook, Inventory: plan.Inventory, Limit: plan.Limit, Environment: plan.Environment, ExtraVars: plan.ExtraVars, Mode: mode})
	if err != nil {
		runTrace.Status = "failed"
		runTrace.Error = err.Error()
		return service.finish(runTrace, request.TraceOut, 1, "", output, now)
	}
	if request.PlanOnly {
		runTrace.Steps = append(runTrace.Steps, model.TraceStep{Name: "plan_only", Mode: string(mode), Command: command, Message: "execution skipped by plan-only mode"})
		runTrace.Status = "planned"
		output.Line("Agent Plan-Only: " + strings.Join(command, " "))
		return service.finish(runTrace, request.TraceOut, 0, "", output, now)
	}

	output.Line("Agent Execute: " + strings.Join(command, " "))
	executionContext, cancelExecution := context.WithTimeout(ctx, timeout)
	result := runner.Run(executionContext, command)
	cancelExecution()
	if result.Stdout != "" {
		output.Stdout(result.Stdout)
	}
	if result.Stderr != "" {
		output.Stderr(result.Stderr)
	}
	logPath, logErr := service.TraceStore.WriteLog(runTrace.RunID, result.Stdout+"\n"+result.Stderr)
	if logPath != "" {
		runTrace.Metadata = map[string]any{"log_path": logPath}
	}
	if logErr != nil {
		runTrace.Error = logErr.Error()
	}
	exitCode := result.ExitCode
	runTrace.Steps = append(runTrace.Steps, model.TraceStep{Name: "execute", Mode: string(mode), Command: command, ExitCode: &exitCode})
	if result.ExitCode == 0 && logErr == nil {
		runTrace.Status = "success"
		output.Line("Agent Verify: execution succeeded")
	} else {
		runTrace.Status = "failed"
		if runTrace.Error == "" {
			runTrace.Error = fmt.Sprintf("execution failed with exit code %d", result.ExitCode)
		}
		output.Line("Agent Verify: " + runTrace.Error)
	}
	responseExitCode := result.ExitCode
	if logErr != nil && responseExitCode == 0 {
		responseExitCode = 1
	}
	return service.finish(runTrace, request.TraceOut, responseExitCode, logPath, output, now)
}

func (service Service) finish(runTrace model.RunTrace, explicitPath string, exitCode int, logPath string, output Sink, now func() time.Time) Response {
	finished := now().UTC()
	runTrace.FinishedAt = &finished
	tracePath, err := service.TraceStore.Write(runTrace, explicitPath)
	if err != nil {
		output.Stderr(err.Error() + "\n")
		return Response{ExitCode: 1, Status: "failed", RunID: runTrace.RunID, LogPath: logPath, Error: err.Error(), Trace: runTrace}
	}
	output.Line("Trace written: " + tracePath)
	return Response{ExitCode: exitCode, Status: runTrace.Status, RunID: runTrace.RunID, TracePath: tracePath, LogPath: logPath, Error: runTrace.Error, Trace: runTrace}
}

func emitPlan(output Sink, plan model.GoalPlan) {
	output.Line("Agent Plan")
	output.Line("Goal: " + plan.Goal)
	output.Line("Playbook: " + valueOr(plan.Playbook, "unknown"))
	output.Line("Environment: " + valueOr(plan.Environment, "unknown"))
	mode := "dry-run"
	if plan.Apply {
		mode = "apply"
	}
	output.Line("Mode: " + mode)
	output.Line(fmt.Sprintf("Confidence: %.2f", plan.Confidence))
	if plan.TemplateID != "" {
		output.Line("Template: " + plan.TemplateID)
	}
	for _, step := range plan.RecommendedSteps {
		output.Line("Recommended: " + step)
	}
	for _, note := range plan.RiskNotes {
		output.Line("Risk note: " + note)
	}
}

func valueOr(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
func boolPointer(value bool) *bool { return &value }
