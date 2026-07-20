package engine

import (
	"reflect"
	"testing"
)

func TestBuildCommandDryRun(t *testing.T) {
	got, err := BuildCommand(CommandRequest{
		Engine:      "ansible",
		Playbook:    "demo.yml",
		Inventory:   "inventory.ini",
		Environment: "dev",
		Mode:        ModeDryRun,
	})
	if err != nil {
		t.Fatalf("BuildCommand() error = %v", err)
	}
	want := []string{"ansible-playbook", "--check", "--diff", "-i", "inventory.ini", "-e", "env=dev", "demo.yml"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildCommand() = %v, want %v", got, want)
	}
}

func TestBuildCommandRequiresPlaybook(t *testing.T) {
	if _, err := BuildCommand(CommandRequest{Mode: ModeDryRun}); err == nil {
		t.Fatal("BuildCommand() error = nil, want error")
	}
}
