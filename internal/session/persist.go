package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// readJSONFile unmarshals path into v. It returns the raw os error, including
// os.IsNotExist, so callers can decide how to handle a missing file.
func readJSONFile(path string, v any) error {
	// #nosec G304 -- path is a caller-constructed state-root metadata file
	// (index.json or usage.json), not user input.
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("parse %s: %w", filepath.Base(path), err)
	}
	return nil
}

// writeJSONFile marshals v as indented JSON and atomically replaces path.
//
// It writes a temporary file in the same directory (with a unique name via
// os.CreateTemp), fsyncs the temporary file, renames it into place, and fsyncs
// the directory so both the data and the rename are durable. The temporary file
// is removed on every error path up to and including a failed rename. Once the
// rename has succeeded there is nothing to remove, so a later failure (opening
// or syncing the directory) leaves the new file in place and returns an error.
func writeJSONFile(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	// Create a unique temporary file in the same directory as path.
	// Use an empty pattern to get the default temp file naming.
	dirPath := filepath.Dir(path)
	tmpFile, err := os.CreateTemp(dirPath, ".tmp")
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	tmp := tmpFile.Name()

	// Write and sync through the single CreateTemp handle to ensure data durability
	// without re-opening. Defer cleanup of the temp file on any error.
	defer func() {
		if tmp != "" {
			_ = os.Remove(tmp)
		}
	}()

	if _, err := tmpFile.WriteString(string(data)); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("write temporary file: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("fsync temporary file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temporary file: %w", err)
	}

	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename temporary file: %w", err)
	}
	// The rename succeeded, so the temporary name no longer exists; disarm the
	// deferred removal. It stays armed on a rename failure.
	tmp = ""

	// Fsync the directory so the rename is durable.
	dirFile, err := os.Open(dirPath) // #nosec G304 -- parent directory of the Kirsch-owned write target (index.json or usage.json under the state root)
	if err != nil {
		return fmt.Errorf("open directory for fsync: %w", err)
	}
	if err := dirFile.Sync(); err != nil {
		_ = dirFile.Close()
		return fmt.Errorf("fsync directory: %w", err)
	}
	if err := dirFile.Close(); err != nil {
		return fmt.Errorf("close directory: %w", err)
	}

	return nil
}
