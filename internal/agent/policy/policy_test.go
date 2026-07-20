package policy

import "testing"

func TestEvaluate(t *testing.T) {
	tests := []struct {
		name    string
		request Request
		allowed bool
	}{
		{
			name:    "dry run needs no approval",
			request: Request{Environment: "prod"},
			allowed: true,
		},
		{
			name:    "apply needs approval",
			request: Request{ExplicitApply: true, ChecksPassed: true, PlanConfidence: 1, MinimumConfidence: .75},
			allowed: false,
		},
		{
			name:    "production needs confirmation",
			request: Request{ExplicitApply: true, Approved: true, ChecksPassed: true, Environment: "prod", RequireProdConfirm: true, PlanConfidence: 1, MinimumConfidence: .75},
			allowed: false,
		},
		{
			name:    "all gates pass",
			request: Request{ExplicitApply: true, Approved: true, ChecksPassed: true, Environment: "prod", RequireProdConfirm: true, ProductionConfirm: "PROD", PlanConfidence: .9, MinimumConfidence: .75},
			allowed: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			decision := Evaluate(test.request)
			if decision.Allowed != test.allowed {
				t.Fatalf("Allowed = %v, want %v; reasons=%v", decision.Allowed, test.allowed, decision.Reasons)
			}
		})
	}
}
