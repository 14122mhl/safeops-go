// Package template contains deterministic release, rollback, and incident templates.
package template

import "strings"

// ChangeTemplate supplies deterministic operational guidance.
type ChangeTemplate struct {
	ID               string
	Name             string
	Keywords         []string
	RecommendedSteps []string
	RiskNotes        []string
}

var templates = []ChangeTemplate{
	{ID: "rollback", Name: "Rollback", Keywords: []string{"rollback", "回滚", "恢复版本"}, RecommendedSteps: []string{"inspect", "check", "dry-run", "approve", "apply", "verify"}, RiskNotes: []string{"confirm the rollback artifact and data compatibility"}},
	{ID: "incident", Name: "Incident triage", Keywords: []string{"incident", "故障", "排障", "恢复服务"}, RecommendedSteps: []string{"collect evidence", "limit scope", "check", "dry-run", "approve", "verify"}, RiskNotes: []string{"preserve evidence and avoid broad changes during triage"}},
	{ID: "release", Name: "Release", Keywords: []string{"release", "deploy", "发布", "部署", "上线"}, RecommendedSteps: []string{"inspect", "check", "dry-run", "approve", "apply", "verify"}, RiskNotes: []string{"use staged rollout and retain rollback evidence"}},
}

// Match returns the first matching template. More specific templates are listed first.
func Match(goal string) (ChangeTemplate, bool) {
	lower := strings.ToLower(goal)
	for _, candidate := range templates {
		for _, keyword := range candidate.Keywords {
			if strings.Contains(lower, strings.ToLower(keyword)) {
				return candidate, true
			}
		}
	}
	return ChangeTemplate{}, false
}
