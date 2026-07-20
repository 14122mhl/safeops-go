// Package analysis performs static playbook risk analysis.
package analysis

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/14122mhl/safeops-go/internal/config"
	"github.com/14122mhl/safeops-go/internal/model"
	"gopkg.in/yaml.v3"
)

var taskSections = []string{"pre_tasks", "tasks", "post_tasks", "handlers"}

var taskMetadata = map[string]bool{
	"name": true, "when": true, "vars": true, "tags": true, "register": true,
	"become": true, "become_user": true, "notify": true, "loop": true,
	"loop_control": true, "ignore_errors": true, "changed_when": true,
	"failed_when": true, "delegate_to": true, "environment": true,
}

// Analyze parses a playbook and assigns configured risk rules to its tasks.
func Analyze(path string, rules map[string]config.RiskRule) (model.PlaybookAnalysis, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return model.PlaybookAnalysis{}, fmt.Errorf("read playbook: %w", err)
	}
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return model.PlaybookAnalysis{}, fmt.Errorf("parse playbook: %w", err)
	}
	result := model.PlaybookAnalysis{Playbook: path, OverallRisk: model.RiskLow}
	if len(root.Content) == 0 || root.Content[0].Kind != yaml.SequenceNode {
		return result, fmt.Errorf("playbook root must be a YAML sequence")
	}
	for playIndex, play := range root.Content[0].Content {
		if play.Kind != yaml.MappingNode {
			continue
		}
		playName := scalar(mappingValue(play, "name"), fmt.Sprintf("play %d", playIndex+1))
		hosts := scalar(mappingValue(play, "hosts"), "unknown")
		for _, section := range taskSections {
			tasks := mappingValue(play, section)
			if tasks == nil || tasks.Kind != yaml.SequenceNode {
				continue
			}
			for taskIndex, task := range tasks.Content {
				item := analyzeTask(task, rules)
				item.PlayIndex = playIndex + 1
				item.PlayName = playName
				item.Hosts = hosts
				item.Section = section
				item.TaskIndex = taskIndex + 1
				result.Tasks = append(result.Tasks, item)
				if riskRank(item.Risk) > riskRank(result.OverallRisk) {
					result.OverallRisk = item.Risk
				}
				if item.Recommendation != "" {
					result.Recommendations = append(result.Recommendations, item.Recommendation)
				}
			}
		}
	}
	result.Recommendations = uniqueSorted(result.Recommendations)
	return result, nil
}

func analyzeTask(task *yaml.Node, rules map[string]config.RiskRule) model.TaskAnalysis {
	item := model.TaskAnalysis{TaskName: "unnamed task", Risk: model.RiskLow, Reason: "no configured risk rule", Recommendation: "review task behavior before apply"}
	if task.Kind != yaml.MappingNode {
		item.Module = "unknown"
		return item
	}
	item.TaskName = scalar(mappingValue(task, "name"), item.TaskName)
	for index := 0; index+1 < len(task.Content); index += 2 {
		key := task.Content[index].Value
		if taskMetadata[key] || strings.HasPrefix(key, "with_") {
			continue
		}
		item.Module = normalizeModule(key)
		break
	}
	if item.Module == "" {
		item.Module = "unknown"
	}
	if rule, ok := rules[item.Module]; ok {
		item.Risk = parseRisk(rule.Risk)
		item.Reason = rule.Reason
		item.Recommendation = rule.Recommendation
	}
	return item
}

func mappingValue(node *yaml.Node, key string) *yaml.Node {
	for index := 0; index+1 < len(node.Content); index += 2 {
		if node.Content[index].Value == key {
			return node.Content[index+1]
		}
	}
	return nil
}

func scalar(node *yaml.Node, fallback string) string {
	if node == nil {
		return fallback
	}
	if node.Kind == yaml.ScalarNode {
		return node.Value
	}
	return fallback
}

func normalizeModule(module string) string {
	parts := strings.Split(module, ".")
	return parts[len(parts)-1]
}

func parseRisk(value string) model.RiskLevel {
	switch strings.ToUpper(value) {
	case "HIGH":
		return model.RiskHigh
	case "MEDIUM":
		return model.RiskMedium
	default:
		return model.RiskLow
	}
}

func riskRank(risk model.RiskLevel) int {
	switch risk {
	case model.RiskHigh:
		return 3
	case model.RiskMedium:
		return 2
	default:
		return 1
	}
}

func uniqueSorted(values []string) []string {
	seen := make(map[string]bool, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}
