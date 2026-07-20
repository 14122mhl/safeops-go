package trace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/14122mhl/safeops-go/internal/model"
)

func TestStoreWritesTraceAndLog(t *testing.T) {
	directory := t.TempDir()
	store := Store{Directory: directory}
	value := model.RunTrace{RunID: "run-1", StartedAt: time.Unix(0, 0).UTC(), Goal: "demo", Status: "planned", Steps: []model.TraceStep{}}
	path, err := store.Write(value, "")
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var decoded model.RunTrace
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("invalid trace JSON: %v", err)
	}
	if decoded.RunID != "run-1" {
		t.Fatalf("RunID = %q", decoded.RunID)
	}
	logPath, err := store.WriteLog("run-1", "output")
	if err != nil {
		t.Fatalf("WriteLog() error = %v", err)
	}
	if filepath.Dir(logPath) != directory {
		t.Fatalf("log path = %q", logPath)
	}
}

func TestNewRunIDIsSortableAndUnique(t *testing.T) {
	now := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
	first := NewRunID(now)
	second := NewRunID(now)
	if first == second {
		t.Fatalf("duplicate run ID %q", first)
	}
	if len(first) < len("20260720T120000.000000000Z") {
		t.Fatalf("run ID = %q", first)
	}
}
