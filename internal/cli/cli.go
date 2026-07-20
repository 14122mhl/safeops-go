// Package cli defines the safeops command-line interface.
package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/14122mhl/safeops-go/internal/agent/policy"
	agentService "github.com/14122mhl/safeops-go/internal/agent/service"
	"github.com/14122mhl/safeops-go/internal/analysis"
	"github.com/14122mhl/safeops-go/internal/check"
	"github.com/14122mhl/safeops-go/internal/config"
	"github.com/14122mhl/safeops-go/internal/engine"
	"github.com/14122mhl/safeops-go/internal/model"
	"github.com/14122mhl/safeops-go/internal/trace"
	"gopkg.in/yaml.v3"
)

// Version is replaced at build time with -ldflags when desired.
var Version = "dev"

// Run parses arguments and returns a process exit code.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	global := flag.NewFlagSet("safeops", flag.ContinueOnError)
	global.SetOutput(stderr)
	configPath := global.String("config", config.DefaultPath, "configuration file path")
	engineName := global.String("engine", "", "automation executor; defaults to configuration")
	global.Usage = func() { printHelp(stdout) }
	if err := global.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	remaining := global.Args()
	if len(remaining) == 0 {
		printHelp(stdout)
		return 0
	}
	explicitConfig := false
	global.Visit(func(item *flag.Flag) {
		if item.Name == "config" {
			explicitConfig = true
		}
	})

	switch remaining[0] {
	case "help", "-h", "--help":
		printHelp(stdout)
		return 0
	case "version":
		fmt.Fprintf(stdout, "safeops %s\n", Version)
		return 0
	case "doctor":
		return runDoctor(stdout)
	case "config":
		return runConfig(remaining[1:], *configPath, explicitConfig, stdout, stderr)
	case "inspect":
		return runInspect(remaining[1:], *configPath, explicitConfig, stdout, stderr)
	case "check":
		return runCheck(ctx, remaining[1:], *configPath, explicitConfig, *engineName, stdout, stderr)
	case "run":
		return runPlaybook(ctx, remaining[1:], *configPath, explicitConfig, *engineName, stdout, stderr)
	case "goal":
		return runGoal(ctx, remaining[1:], *configPath, explicitConfig, *engineName, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", remaining[0])
		printHelp(stderr)
		return 2
	}
}

type consoleSink struct{ stdout, stderr io.Writer }

func (sink consoleSink) Line(value string)   { fmt.Fprintln(sink.stdout, value) }
func (sink consoleSink) Stdout(value string) { fmt.Fprint(sink.stdout, value) }
func (sink consoleSink) Stderr(value string) { fmt.Fprint(sink.stderr, value) }

func runGoal(ctx context.Context, args []string, configPath string, explicitConfig bool, engineName string, stdout, stderr io.Writer) int {
	request, err := parseGoalRequest(args, stderr)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	cfg, ok := loadRuntimeConfig(configPath, explicitConfig, stderr)
	if !ok {
		return 1
	}
	request.Engine = effectiveEngine(engineName, cfg)
	response := (agentService.Service{Config: cfg, Runner: engine.ExecRunner{}, TraceStore: trace.Store{}}).Run(ctx, request, consoleSink{stdout: stdout, stderr: stderr})
	return response.ExitCode
}

func parseGoalRequest(args []string, stderr io.Writer) (agentService.Request, error) {
	var request agentService.Request
	var dryRun bool
	goalEnd := 0
	for goalEnd < len(args) && !strings.HasPrefix(args[goalEnd], "-") {
		goalEnd++
	}
	if goalEnd == 0 {
		return request, fmt.Errorf("usage: safeops goal <goal...> [options]")
	}
	request.Goal = strings.Join(args[:goalEnd], " ")
	flags := flag.NewFlagSet("goal", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&request.Playbook, "playbook", "", "playbook path; overrides goal inference")
	flags.StringVar(&request.Inventory, "inventory", "", "inventory file or directory")
	flags.StringVar(&request.Inventory, "i", "", "inventory file or directory")
	flags.StringVar(&request.Environment, "env", "", "environment name")
	flags.StringVar(&request.Limit, "limit", "", "limit target hosts")
	flags.Var((*stringList)(&request.ExtraVars), "extra-var", "extra variable; repeatable")
	flags.Var((*stringList)(&request.ExtraVars), "e", "extra variable; repeatable")
	flags.BoolVar(&request.ExplicitApply, "apply", false, "request real execution after policy gates")
	flags.BoolVar(&dryRun, "dry-run", false, "force non-mutating execution mode")
	flags.BoolVar(&request.Approved, "approve", false, "approve apply policy gate")
	flags.BoolVar(&request.PlanOnly, "plan-only", false, "plan, check, and preview without execution")
	flags.StringVar(&request.ProductionConfirm, "confirm", "", "production confirmation value")
	flags.StringVar(&request.TraceOut, "trace-out", "", "write trace to a specific path")
	flags.DurationVar(&request.Timeout, "timeout", 10*time.Minute, "check and execution timeout")
	if err := flags.Parse(args[goalEnd:]); err != nil {
		return request, err
	}
	if request.ExplicitApply && dryRun {
		return request, fmt.Errorf("--apply and --dry-run cannot be used together")
	}
	if flags.NArg() != 0 {
		return request, fmt.Errorf("unexpected goal arguments: %s", strings.Join(flags.Args(), " "))
	}
	return request, nil
}

type playbookOptions struct {
	playbook, inventory, limit, environment, report string
	extraVars                                       []string
	apply, approve                                  bool
	confirm                                         string
	timeout                                         time.Duration
}

type stringList []string

func (values *stringList) String() string         { return strings.Join(*values, ",") }
func (values *stringList) Set(value string) error { *values = append(*values, value); return nil }

func parsePlaybookOptions(name string, args []string, allowApply bool, stderr io.Writer) (playbookOptions, error) {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(stderr)
	var options playbookOptions
	flags.StringVar(&options.inventory, "inventory", "", "inventory file or directory")
	flags.StringVar(&options.inventory, "i", "", "inventory file or directory")
	flags.StringVar(&options.limit, "limit", "", "limit target hosts")
	flags.StringVar(&options.environment, "env", "", "environment name")
	flags.Var((*stringList)(&options.extraVars), "extra-var", "extra variable; repeatable")
	flags.Var((*stringList)(&options.extraVars), "e", "extra variable; repeatable")
	flags.StringVar(&options.report, "report", "", "write JSON check report")
	flags.DurationVar(&options.timeout, "timeout", 10*time.Minute, "execution timeout")
	if allowApply {
		flags.BoolVar(&options.apply, "apply", false, "request a real change; default is dry-run")
		flags.BoolVar(&options.approve, "approve", false, "approve apply policy gate")
		flags.StringVar(&options.confirm, "confirm", "", "production confirmation value")
	}
	parseArgs := args
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		options.playbook = args[0]
		parseArgs = args[1:]
	}
	if err := flags.Parse(parseArgs); err != nil {
		return options, err
	}
	if options.playbook == "" && flags.NArg() == 1 {
		options.playbook = flags.Arg(0)
	}
	if options.playbook == "" || flags.NArg() > 1 {
		return options, fmt.Errorf("usage: safeops %s <playbook> [options]", name)
	}
	return options, nil
}

func loadRuntimeConfig(path string, explicit bool, stderr io.Writer) (config.Config, bool) {
	cfg, err := config.Load(path, explicit)
	if err != nil {
		fmt.Fprintf(stderr, "configuration: %v\n", err)
		return config.Config{}, false
	}
	return cfg, true
}

func runInspect(args []string, configPath string, explicitConfig bool, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "usage: safeops inspect <playbook>")
		return 2
	}
	cfg, ok := loadRuntimeConfig(configPath, explicitConfig, stderr)
	if !ok {
		return 1
	}
	result, err := analysis.Analyze(args[0], cfg.Risk)
	if err != nil {
		fmt.Fprintf(stderr, "inspect: %v\n", err)
		return 1
	}
	emitAnalysis(stdout, result)
	return 0
}

func runCheck(ctx context.Context, args []string, configPath string, explicitConfig bool, engineName string, stdout, stderr io.Writer) int {
	options, err := parsePlaybookOptions("check", args, false, stderr)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	cfg, ok := loadRuntimeConfig(configPath, explicitConfig, stderr)
	if !ok {
		return 1
	}
	if options.environment == "" {
		options.environment = cfg.Settings.DefaultEnv
	}
	results := performChecks(ctx, options, cfg, engineName)
	emitChecks(stdout, results)
	if err := writeReport(options.report, results); err != nil {
		fmt.Fprintf(stderr, "report: %v\n", err)
		return 1
	}
	if check.HasFailures(results) {
		return 1
	}
	return 0
}

func runPlaybook(ctx context.Context, args []string, configPath string, explicitConfig bool, engineName string, stdout, stderr io.Writer) int {
	options, err := parsePlaybookOptions("run", args, true, stderr)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	cfg, ok := loadRuntimeConfig(configPath, explicitConfig, stderr)
	if !ok {
		return 1
	}
	if options.environment == "" {
		options.environment = cfg.Settings.DefaultEnv
	}
	results := performChecks(ctx, options, cfg, engineName)
	emitChecks(stdout, results)
	if err := writeReport(options.report, results); err != nil {
		fmt.Fprintf(stderr, "report: %v\n", err)
		return 1
	}
	decision := policy.Evaluate(policy.Request{
		ExplicitApply: options.apply, Approved: options.approve, Environment: options.environment,
		ProductionConfirm: options.confirm, RequireProdConfirm: cfg.Settings.RequireProdConfirm,
		PlanConfidence: 1, MinimumConfidence: cfg.Settings.MinGoalConfidenceToApply,
		ChecksPassed: !check.HasFailures(results),
	})
	for _, reason := range decision.Reasons {
		fmt.Fprintf(stdout, "[POLICY] %s\n", reason)
	}
	if !decision.Allowed {
		return 1
	}
	mode := engine.ModeDryRun
	if options.apply {
		mode = engine.ModeApply
	}
	command, err := engine.BuildCommand(engine.CommandRequest{Engine: effectiveEngine(engineName, cfg), Playbook: options.playbook, Inventory: options.inventory, Limit: options.limit, Environment: options.environment, ExtraVars: options.extraVars, Mode: mode})
	if err != nil {
		fmt.Fprintf(stderr, "run: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Execution mode: %s\nCommand: %s\n", mode, strings.Join(command, " "))
	executionContext, cancel := context.WithTimeout(ctx, options.timeout)
	defer cancel()
	result := (engine.ExecRunner{}).Run(executionContext, command)
	if result.Stdout != "" {
		fmt.Fprint(stdout, result.Stdout)
	}
	if result.Stderr != "" {
		fmt.Fprint(stderr, result.Stderr)
	}
	return result.ExitCode
}

func performChecks(ctx context.Context, options playbookOptions, cfg config.Config, engineName string) []model.CheckResult {
	validationContext, cancel := context.WithTimeout(ctx, options.timeout)
	defer cancel()
	return check.Preflight(validationContext, check.Request{Playbook: options.playbook, Inventory: options.inventory, Limit: options.limit, Environment: options.environment, ExtraVars: options.extraVars, Engine: effectiveEngine(engineName, cfg)}, cfg.Risk, engine.ExecRunner{})
}

func effectiveEngine(name string, cfg config.Config) string {
	if name != "" {
		return name
	}
	return cfg.Settings.DefaultEngine
}

func emitAnalysis(writer io.Writer, result model.PlaybookAnalysis) {
	fmt.Fprintf(writer, "Playbook: %s\nOverall Risk: %s\nTasks: %d\n", result.Playbook, result.OverallRisk, len(result.Tasks))
	for _, task := range result.Tasks {
		fmt.Fprintf(writer, "- Play %d (%s) %s task %d: %s\n  Hosts: %s\n  Module: %s\n  Risk: %s\n  Reason: %s\n  Recommendation: %s\n", task.PlayIndex, task.PlayName, task.Section, task.TaskIndex, task.TaskName, task.Hosts, task.Module, task.Risk, task.Reason, task.Recommendation)
	}
}

func emitChecks(writer io.Writer, results []model.CheckResult) {
	for _, result := range results {
		fmt.Fprintf(writer, "[%s] %s: %s\n", result.Status, result.Name, result.Message)
		if result.Remediation != "" {
			fmt.Fprintf(writer, "       remediation: %s\n", result.Remediation)
		}
	}
}

func writeReport(path string, results []model.CheckResult) error {
	if path == "" {
		return nil
	}
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

func runDoctor(stdout io.Writer) int {
	for _, result := range check.Doctor() {
		fmt.Fprintf(stdout, "[%s] %s: %s\n", result.Status, result.Name, result.Message)
		if result.Remediation != "" {
			fmt.Fprintf(stdout, "       remediation: %s\n", result.Remediation)
		}
	}
	return 0
}

func runConfig(args []string, path string, explicit bool, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "usage: safeops [--config path] config <show|init>")
		return 2
	}
	switch args[0] {
	case "show":
		cfg, err := config.Load(path, explicit)
		if err != nil {
			fmt.Fprintf(stderr, "config show: %v\n", err)
			return 1
		}
		data, err := yaml.Marshal(cfg.Masked())
		if err != nil {
			fmt.Fprintf(stderr, "config show: encode configuration: %v\n", err)
			return 1
		}
		_, _ = stdout.Write(data)
		return 0
	case "init":
		if err := config.WriteDefault(path); err != nil {
			fmt.Fprintf(stderr, "config init: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "configuration written: %s\n", path)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown config command %q\n", args[0])
		return 2
	}
}

func printHelp(writer io.Writer) {
	commands := []string{
		"check          run local, static, and Ansible preflight checks",
		"config init   write a conservative default configuration",
		"config show   print effective configuration with secrets masked",
		"doctor        inspect local Go and Ansible tooling",
		"goal           evaluate a natural-language goal through the Agent Kernel",
		"inspect        analyze playbook tasks and configured risk",
		"run            preflight then dry-run; requires --apply --approve for changes",
		"version       print build version",
	}
	sort.Strings(commands)
	fmt.Fprintln(writer, "safeops: safe change agent platform (Go rewrite)")
	fmt.Fprintln(writer, "")
	fmt.Fprintln(writer, "Usage:")
	fmt.Fprintln(writer, "  safeops [--config path] [--engine name] <command>")
	fmt.Fprintln(writer, "")
	fmt.Fprintln(writer, "Commands:")
	for _, command := range commands {
		fmt.Fprintf(writer, "  %s\n", command)
	}
	fmt.Fprintln(writer, "")
	fmt.Fprintln(writer, "Safety: real changes will require an explicit --apply control; semantic text never grants permission.")
}
