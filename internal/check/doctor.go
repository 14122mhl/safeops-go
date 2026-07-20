// Package check implements environment and change preflight checks.
package check

import (
	"os/exec"
	"runtime"

	"github.com/14122mhl/safeops-go/internal/model"
)

// Doctor inspects the local runtime without changing system state.
func Doctor() []model.CheckResult {
	results := []model.CheckResult{
		{Name: "go", Status: model.CheckPass, Message: runtime.Version()},
	}
	if path, err := exec.LookPath("ansible-playbook"); err == nil {
		results = append(results, model.CheckResult{Name: "ansible", Status: model.CheckPass, Message: path})
	} else {
		results = append(results, model.CheckResult{
			Name:        "ansible",
			Status:      model.CheckWarn,
			Message:     "ansible-playbook not found; execution checks will be unavailable",
			Remediation: "install Ansible before validating or running playbooks",
		})
	}
	return results
}
