package analysis

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/14122mhl/safeops-go/internal/config"
	"github.com/14122mhl/safeops-go/internal/model"
)

func TestAnalyzeAssignsRiskToQualifiedModules(t *testing.T) {
	path := filepath.Join(t.TempDir(), "playbook.yml")
	content := "---\n" +
		"- name: deploy\n" +
		"  hosts: web\n" +
		"  tasks:\n" +
		"    - name: render configuration\n" +
		"      ansible.builtin.template:\n" +
		"        src: app.conf\n" +
		"        dest: /etc/app.conf\n" +
		"    - name: restart service\n" +
		"      ansible.builtin.service:\n" +
		"        name: app\n" +
		"        state: restarted\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := Analyze(path, config.Default().Risk)
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if result.OverallRisk != model.RiskHigh {
		t.Fatalf("OverallRisk = %s, want HIGH", result.OverallRisk)
	}
	if len(result.Tasks) != 2 {
		t.Fatalf("tasks = %d, want 2", len(result.Tasks))
	}
	if result.Tasks[0].Module != "template" || result.Tasks[0].Risk != model.RiskMedium {
		t.Fatalf("first task = %+v", result.Tasks[0])
	}
	if result.Tasks[1].Module != "service" || result.Tasks[1].Risk != model.RiskHigh {
		t.Fatalf("second task = %+v", result.Tasks[1])
	}
}

func TestAnalyzeRejectsMappingRoot(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid.yml")
	if err := os.WriteFile(path, []byte("hosts: all\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Analyze(path, config.Default().Risk); err == nil {
		t.Fatal("Analyze() error = nil, want root error")
	}
}
