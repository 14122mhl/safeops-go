// Package model defines the domain contracts shared by safeops components.
package model

import "time"

// CheckStatus is the outcome of a safety check.
type CheckStatus string

const (
	CheckPass CheckStatus = "PASS"
	CheckWarn CheckStatus = "WARN"
	CheckFail CheckStatus = "FAIL"
)

// RiskLevel describes the potential impact of an operation.
type RiskLevel string

const (
	RiskLow    RiskLevel = "LOW"
	RiskMedium RiskLevel = "MEDIUM"
	RiskHigh   RiskLevel = "HIGH"
)

// CheckResult captures one preflight or validation result.
type CheckResult struct {
	Name        string      `json:"name" yaml:"name"`
	Status      CheckStatus `json:"status" yaml:"status"`
	Message     string      `json:"message" yaml:"message"`
	Remediation string      `json:"remediation,omitempty" yaml:"remediation,omitempty"`
}

// GoalPlan is a normalized change request. Apply can only be set from an
// explicit operator control; semantic hints must never authorize execution.
type GoalPlan struct {
	Goal          string   `json:"goal"`
	Playbook      string   `json:"playbook,omitempty"`
	Inventory     string   `json:"inventory,omitempty"`
	Environment   string   `json:"env,omitempty"`
	Limit         string   `json:"limit,omitempty"`
	ExtraVars     []string `json:"extra_vars,omitempty"`
	Apply         bool     `json:"apply"`
	Confidence    float64  `json:"confidence"`
	MissingFields []string `json:"missing_fields,omitempty"`
	Notes         []string `json:"notes,omitempty"`
}

// TraceStep records one stage of the agent workflow.
type TraceStep struct {
	Name     string        `json:"name"`
	Allowed  *bool         `json:"allowed,omitempty"`
	Reasons  []string      `json:"reasons,omitempty"`
	Checks   []CheckResult `json:"checks,omitempty"`
	ExitCode *int          `json:"exit_code,omitempty"`
}

// RunTrace is the auditable record of a safeops request.
type RunTrace struct {
	RunID      string         `json:"run_id"`
	StartedAt  time.Time      `json:"started_at"`
	FinishedAt *time.Time     `json:"finished_at,omitempty"`
	Goal       string         `json:"goal"`
	Status     string         `json:"status"`
	Plan       *GoalPlan      `json:"plan,omitempty"`
	Steps      []TraceStep    `json:"steps"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}
