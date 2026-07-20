// Package trace persists auditable safeops run records.
package trace

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/14122mhl/safeops-go/internal/model"
)

// Store writes trace and execution evidence beneath a run directory.
type Store struct{ Directory string }

// Summary is the bounded run information displayed by local clients.
type Summary struct {
	RunID       string          `json:"run_id"`
	Status      string          `json:"status"`
	Goal        string          `json:"goal"`
	StartedAt   time.Time       `json:"started_at"`
	Playbook    string          `json:"playbook,omitempty"`
	Environment string          `json:"env,omitempty"`
	Risk        model.RiskLevel `json:"risk,omitempty"`
	Path        string          `json:"path"`
}

// NewRunID returns a sortable UTC ID with random collision protection.
func NewRunID(now time.Time) string {
	random := make([]byte, 4)
	if _, err := rand.Read(random); err != nil {
		return now.UTC().Format("20060102T150405.000000000Z")
	}
	return now.UTC().Format("20060102T150405.000000000Z") + "-" + hex.EncodeToString(random)
}

// Write atomically persists a JSON trace and returns its path.
func (store Store) Write(value model.RunTrace, explicitPath string) (string, error) {
	path := explicitPath
	if path == "" {
		path = filepath.Join(store.directory(), value.RunID+".json")
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode trace: %w", err)
	}
	if err := writeAtomic(path, append(data, '\n'), 0o600); err != nil {
		return "", fmt.Errorf("write trace: %w", err)
	}
	return path, nil
}

// WriteLog persists combined command output for later diagnosis.
func (store Store) WriteLog(runID, content string) (string, error) {
	path := filepath.Join(store.directory(), runID+".log")
	if err := writeAtomic(path, []byte(content), 0o600); err != nil {
		return "", fmt.Errorf("write log: %w", err)
	}
	return path, nil
}

// Recent returns newest valid traces and skips malformed or unrelated files.
func (store Store) Recent(limit int) ([]Summary, error) {
	if limit <= 0 {
		limit = 8
	}
	entries, err := os.ReadDir(store.directory())
	if os.IsNotExist(err) {
		return []Summary{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read trace directory: %w", err)
	}
	type candidate struct {
		path     string
		modified time.Time
	}
	candidates := make([]candidate, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || strings.ToLower(filepath.Ext(entry.Name())) != ".json" {
			continue
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			continue
		}
		candidates = append(candidates, candidate{path: filepath.Join(store.directory(), entry.Name()), modified: info.ModTime()})
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].modified.After(candidates[j].modified) })
	capacity := limit
	if len(candidates) < capacity {
		capacity = len(candidates)
	}
	result := make([]Summary, 0, capacity)
	for _, candidate := range candidates {
		if len(result) >= limit {
			break
		}
		data, readErr := os.ReadFile(candidate.path)
		if readErr != nil {
			continue
		}
		var value model.RunTrace
		if json.Unmarshal(data, &value) != nil || value.RunID == "" {
			continue
		}
		summary := Summary{RunID: value.RunID, Status: value.Status, Goal: value.Goal, StartedAt: value.StartedAt, Path: candidate.path}
		if value.Plan != nil {
			summary.Playbook = value.Plan.Playbook
			summary.Environment = value.Plan.Environment
		}
		if value.Analysis != nil {
			summary.Risk = value.Analysis.OverallRisk
		}
		result = append(result, summary)
	}
	return result, nil
}

func (store Store) directory() string {
	if store.Directory != "" {
		return store.Directory
	}
	return filepath.Join(".safeops", "runs")
}

func writeAtomic(path string, data []byte, mode os.FileMode) error {
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(directory, ".trace-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(mode); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}
