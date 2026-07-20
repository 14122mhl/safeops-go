// Package planner turns a user goal and trusted controls into a normalized plan.
package planner

import (
	"regexp"
	"strings"

	"github.com/14122mhl/safeops-go/internal/model"
	"github.com/14122mhl/safeops-go/internal/provider"
)

var playbookPattern = regexp.MustCompile(`([\w./-]+\.ya?ml)`)
var inventoryPattern = regexp.MustCompile(`([\w./-]+\.ini)`)
var englishTokenPattern = regexp.MustCompile(`[a-z]+`)

// Request separates untrusted semantic text from trusted operator controls.
type Request struct {
	Goal                                    string
	Playbook                                string
	Inventory                               string
	Environment                             string
	Limit                                   string
	ExtraVars                               []string
	ExplicitApply                           bool
	DefaultEnv                              string
	Hints                                   *provider.Hints
	PlaybookCandidates, InventoryCandidates []string
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
		if request.Hints != nil && isCandidate(request.Hints.Playbook, request.PlaybookCandidates) {
			plan.Playbook = request.Hints.Playbook
			plan.Confidence += 0.35
			plan.Notes = append(plan.Notes, "playbook suggested by semantic provider")
		} else if match := playbookPattern.FindStringSubmatch(request.Goal); len(match) > 1 {
			plan.Playbook = match[1]
			plan.Confidence += 0.35
			plan.Notes = append(plan.Notes, "playbook inferred from goal")
		}
	} else {
		plan.Confidence += 0.4
		plan.Notes = append(plan.Notes, "playbook from explicit control")
	}
	if plan.Inventory == "" {
		if request.Hints != nil && isCandidate(request.Hints.Inventory, request.InventoryCandidates) {
			plan.Inventory = request.Hints.Inventory
			plan.Confidence += 0.1
			plan.Notes = append(plan.Notes, "inventory suggested by semantic provider")
		} else if match := inventoryPattern.FindStringSubmatch(request.Goal); len(match) > 1 {
			plan.Inventory = match[1]
			plan.Confidence += 0.1
			plan.Notes = append(plan.Notes, "inventory inferred from goal")
		}
	} else {
		plan.Confidence += 0.1
		plan.Notes = append(plan.Notes, "inventory from explicit control")
	}
	if plan.Environment == "" {
		if request.Hints != nil {
			plan.Environment = normalizeEnvironment(request.Hints.Environment)
		}
		if plan.Environment != "" {
			plan.Confidence += 0.15
			plan.Notes = append(plan.Notes, "environment suggested by semantic provider")
		} else {
			plan.Environment = inferEnvironment(request.Goal)
		}
		if plan.Environment == "" {
			plan.Environment = request.DefaultEnv
			plan.Confidence += 0.05
			plan.Notes = append(plan.Notes, "environment from default settings")
		} else if !contains(plan.Notes, "environment suggested by semantic provider") {
			plan.Confidence += 0.15
			plan.Notes = append(plan.Notes, "environment inferred from goal")
		}
	} else {
		plan.Confidence += 0.2
	}
	if request.Hints != nil {
		if plan.Limit == "" {
			plan.Limit = request.Hints.Limit
		}
		plan.ExtraVars = unique(append(plan.ExtraVars, request.Hints.ExtraVars...))
		plan.Notes = append(plan.Notes, request.Hints.Notes...)
		plan.Notes = append(plan.Notes, request.Hints.Reasoning...)
		plan.RecommendedSteps = append(plan.RecommendedSteps, request.Hints.RecommendedSteps...)
		plan.RiskNotes = append(plan.RiskNotes, request.Hints.RiskNotes...)
		plan.Confidence += request.Hints.Confidence * 0.05
		if request.Hints.ApplyIntent && !request.ExplicitApply {
			plan.Notes = append(plan.Notes, "semantic apply intent ignored; explicit --apply is required")
		}
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

func isCandidate(value string, candidates []string) bool {
	if value == "" {
		return false
	}
	for _, candidate := range candidates {
		if value == candidate {
			return true
		}
	}
	return false
}
func normalizeEnvironment(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	aliases := map[string]string{"dev": "dev", "development": "dev", "test": "test", "testing": "test", "stage": "stage", "staging": "stage", "prod": "prod", "production": "prod"}
	return aliases[value]
}
func contains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
func unique(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	return result
}

func inferEnvironment(goal string) string {
	lower := strings.ToLower(goal)
	englishAliases := map[string]string{
		"production": "prod", "prod": "prod",
		"staging": "stage", "stage": "stage",
		"testing": "test", "test": "test",
		"development": "dev", "develop": "dev", "dev": "dev",
	}
	for _, token := range englishTokenPattern.FindAllString(lower, -1) {
		if environment, ok := englishAliases[token]; ok {
			return environment
		}
	}
	chineseAliases := []struct{ token, value string }{{"生产", "prod"}, {"预发", "stage"}, {"测试", "test"}, {"开发", "dev"}}
	for _, alias := range chineseAliases {
		if strings.Contains(goal, alias.token) {
			return alias.value
		}
	}
	return ""
}
