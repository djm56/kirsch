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
// It writes a temporary file in the same directory, fsyncs the temporary file,
// renames it into place, and fsyncs the directory so both the data and the
// rename are durable. If the rename fails, the temporary file is removed on a
// best-effort basis.
func writeJSONFile(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write temporary file: %w", err)
	}

	// Fsync the temporary file so the data is on disk before the rename.
	tmpFile, err := os.Open(tmp)
	if err != nil {
		return fmt.Errorf("open temporary file for fsync: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("fsync temporary file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temporary file: %w", err)
	}

	if err := os.Rename(tmp, path); err != nil {
		// Best-effort cleanup of the temporary file on rename failure.
		_ = os.Remove(tmp)
		return fmt.Errorf("rename temporary file: %w", err)
	}

	// Fsync the directory so the rename is durable.
	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		return fmt.Errorf("open directory for fsync: %w", err)
	}
	if err := dir.Sync(); err != nil {
		_ = dir.Close()
		return fmt.Errorf("fsync directory: %w", err)
	}
	if err := dir.Close(); err != nil {
		return fmt.Errorf("close directory: %w", err)
	}

	return nil
}
