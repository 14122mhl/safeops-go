// Package provider defines optional semantic reasoning contracts.
package provider

import "context"

// Request contains bounded context supplied to a reasoning provider.
type Request struct {
	Goal                string
	PlaybookCandidates  []string
	InventoryCandidates []string
	TemplateID          string
	RetrievedContext    string
}

// Hints are untrusted semantic suggestions. ApplyIntent must never authorize execution.
type Hints struct {
	Playbook         string   `json:"playbook"`
	Inventory        string   `json:"inventory"`
	Environment      string   `json:"environment"`
	Limit            string   `json:"limit"`
	ExtraVars        []string `json:"extra_vars"`
	ApplyIntent      bool     `json:"apply_intent"`
	Confidence       float64  `json:"confidence"`
	Notes            []string `json:"notes"`
	RecommendedSteps []string `json:"recommended_steps"`
	Reasoning        []string `json:"reasoning"`
	RiskNotes        []string `json:"risk_notes"`
	MissingFields    []string `json:"missing_fields"`
}

// GoalParser parses a goal without executing tools or granting permission.
type GoalParser interface {
	ParseGoal(context.Context, Request) (Hints, error)
}
