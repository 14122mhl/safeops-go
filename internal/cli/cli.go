// Package cli defines the safeops command-line interface.
package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"sort"

	"github.com/14122mhl/safeops-go/internal/check"
	"github.com/14122mhl/safeops-go/internal/config"
	"gopkg.in/yaml.v3"
)

// Version is replaced at build time with -ldflags when desired.
var Version = "dev"

// Run parses arguments and returns a process exit code.
func Run(_ context.Context, args []string, stdout, stderr io.Writer) int {
	global := flag.NewFlagSet("safeops", flag.ContinueOnError)
	global.SetOutput(stderr)
	configPath := global.String("config", config.DefaultPath, "configuration file path")
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
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", remaining[0])
		printHelp(stderr)
		return 2
	}
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
		"config init   write a conservative default configuration",
		"config show   print effective configuration with secrets masked",
		"doctor        inspect local Go and Ansible tooling",
		"version       print build version",
	}
	sort.Strings(commands)
	fmt.Fprintln(writer, "safeops: safe change agent platform (Go rewrite)")
	fmt.Fprintln(writer, "")
	fmt.Fprintln(writer, "Usage:")
	fmt.Fprintln(writer, "  safeops [--config path] <command>")
	fmt.Fprintln(writer, "")
	fmt.Fprintln(writer, "Commands:")
	for _, command := range commands {
		fmt.Fprintf(writer, "  %s\n", command)
	}
	fmt.Fprintln(writer, "")
	fmt.Fprintln(writer, "Safety: real changes will require an explicit --apply control; semantic text never grants permission.")
}
