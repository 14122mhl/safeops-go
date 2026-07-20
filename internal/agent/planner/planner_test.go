package planner

import "testing"

func TestBuildDefaultsSemanticReleaseToDryRun(t *testing.T) {
	plan := Build(Request{Goal: "安全发布 demo.yml 到 prod", DefaultEnv: "dev"})
	if plan.Apply {
		t.Fatal("Apply = true without explicit operator control")
	}
	if plan.Playbook != "demo.yml" {
		t.Fatalf("Playbook = %q, want demo.yml", plan.Playbook)
	}
	if plan.Environment != "prod" {
		t.Fatalf("Environment = %q, want prod", plan.Environment)
	}
}

func TestBuildAllowsExplicitApplyRequest(t *testing.T) {
	plan := Build(Request{
		Goal:          "deploy demo.yml to dev",
		ExplicitApply: true,
		DefaultEnv:    "dev",
	})
	if !plan.Apply {
		t.Fatal("Apply = false with explicit apply control")
	}
}

func TestBuildMarksMissingPlaybook(t *testing.T) {
	plan := Build(Request{Goal: "deploy to dev", DefaultEnv: "dev"})
	if len(plan.MissingFields) != 1 || plan.MissingFields[0] != "playbook" {
		t.Fatalf("MissingFields = %v, want [playbook]", plan.MissingFields)
	}
}
