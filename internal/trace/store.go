// Package trace persists auditable safeops run records.
package trace

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/14122mhl/safeops-go/internal/model"
)

// Store writes trace and execution evidence beneath a run directory.
type Store struct{ Directory string }

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
