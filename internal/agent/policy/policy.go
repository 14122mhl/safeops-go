// Package policy enforces execution authorization independently of semantic reasoning.
package policy

import "fmt"

// Request contains trusted operator controls and evaluated plan facts.
type Request struct {
	ExplicitApply      bool
	Approved           bool
	Environment        string
	ProductionConfirm  string
	RequireProdConfirm bool
	PlanConfidence     float64
	MinimumConfidence  float64
	ChecksPassed       bool
}

// Decision is the immutable result of policy evaluation.
type Decision struct {
	Allowed bool
	Reasons []string
}

// Evaluate permits dry-run by default and requires every apply gate to pass.
func Evaluate(request Request) Decision {
	if !request.ExplicitApply {
		return Decision{Allowed: true, Reasons: []string{"dry-run mode; approval not required"}}
	}
	decision := Decision{Allowed: true}
	deny := func(reason string) {
		decision.Allowed = false
		decision.Reasons = append(decision.Reasons, reason)
	}
	if !request.Approved {
		deny("apply requires explicit operator approval")
	}
	if !request.ChecksPassed {
		deny("apply requires all preflight checks to pass")
	}
	if request.PlanConfidence < request.MinimumConfidence {
		deny(fmt.Sprintf("plan confidence %.2f is below threshold %.2f", request.PlanConfidence, request.MinimumConfidence))
	}
	isProduction := request.Environment == "prod" || request.Environment == "production"
	if isProduction && request.RequireProdConfirm && request.ProductionConfirm != "PROD" {
		deny("production apply requires confirmation PROD")
	}
	if decision.Allowed {
		decision.Reasons = append(decision.Reasons, "all apply gates passed")
	}
	return decision
}
