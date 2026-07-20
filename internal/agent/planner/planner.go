// Package planner turns a user goal and trusted controls into a normalized plan.
package planner

import (
	"regexp"
	"strings"

	"github.com/14122mhl/safeops-go/internal/model"
)

var playbookPattern = regexp.MustCompile(`([\w./-]+\.ya?ml)`)

// Request separates untrusted semantic text from trusted operator controls.
type Request struct {
	Goal          string
	Playbook      string
	Inventory     string
	Environment   string
	Limit         string
	ExtraVars     []string
	ExplicitApply bool
	DefaultEnv    string
}

// Build performs deterministic local parsing. ExplicitApply is the only input
// allowed to set GoalPlan.Apply.
func Build(request Request) model.GoalPlan {
	plan := model.GoalPlan{
		Goal:        request.Goal,
		Playbook:    request.Playbook,
		Inventory:   request.Inventory,
		Environment: request.Environment,
		Limit:       request.Limit,
		ExtraVars:   append([]string(nil), request.ExtraVars...),
		Apply:       request.ExplicitApply,
		Confidence:  0.1,
	}
	if plan.Playbook == "" {
		if match := playbookPattern.FindStringSubmatch(request.Goal); len(match) > 1 {
			plan.Playbook = match[1]
			plan.Confidence += 0.35
			plan.Notes = append(plan.Notes, "playbook inferred from goal")
		}
	} else {
		plan.Confidence += 0.4
		plan.Notes = append(plan.Notes, "playbook from explicit control")
	}
	if plan.Environment == "" {
		plan.Environment = inferEnvironment(request.Goal)
		if plan.Environment == "" {
			plan.Environment = request.DefaultEnv
			plan.Confidence += 0.05
			plan.Notes = append(plan.Notes, "environment from default settings")
		} else {
			plan.Confidence += 0.15
			plan.Notes = append(plan.Notes, "environment inferred from goal")
		}
	} else {
		plan.Confidence += 0.2
	}
	if plan.Playbook == "" {
		plan.MissingFields = append(plan.MissingFields, "playbook")
	}
	if request.ExplicitApply {
		plan.Confidence += 0.1
		plan.Notes = append(plan.Notes, "apply requested by explicit operator control")
	} else {
		plan.Notes = append(plan.Notes, "mode defaults to dry-run; semantic text cannot authorize apply")
	}
	if plan.Confidence > 1 {
		plan.Confidence = 1
	}
	return plan
}

func inferEnvironment(goal string) string {
	lower := strings.ToLower(goal)
	aliases := []struct{ token, value string }{
		{"production", "prod"}, {"prod", "prod"}, {"生产", "prod"},
		{"staging", "stage"}, {"stage", "stage"}, {"预发", "stage"},
		{"testing", "test"}, {"test", "test"}, {"测试", "test"},
		{"development", "dev"}, {"develop", "dev"}, {"dev", "dev"}, {"开发", "dev"},
	}
	for _, alias := range aliases {
		if strings.Contains(lower, alias.token) {
			return alias.value
		}
	}
	return ""
}
