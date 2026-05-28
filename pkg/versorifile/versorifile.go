/*
 * Copyright (c) 2026 Versori Group Inc
 *
 * Use of this software is governed by the Business Source License 1.1
 * included in the LICENSE file at the root of this repository.
 *
 * Change Date: 2030-03-01
 * Change License: Apache License, Version 2.0
 *
 * As of the Change Date, in accordance with the Business Source License,
 * use of this software will be governed by the Apache License, Version 2.0.
 */

// Package versorifile reads and writes the per-project `.versori` JSON file
// written by `versori projects sync`. The file pins a project ID and the
// context that was active when the project was synced, so any subsequent
// CLI invocation from inside the directory can default both project and
// context without explicit flags.
//
// Lives outside `pkg/cmd/flags` so `pkg/cmd/config` (which the flags package
// itself depends on) can read it without producing an import cycle.
package versorifile

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// VersoriFile is the shape of the .versori file in a synced project directory.
type VersoriFile struct {
	ProjectId string `json:"project_id"`
	Context   string `json:"context"`
}

// Read parses a .versori file at the given absolute path. Returns the standard
// fs error wrapper on miss, so callers can branch on os.IsNotExist.
func Read(path string) (*VersoriFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var v VersoriFile
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}

	return &v, nil
}

// Write persists the .versori file at the given absolute path.
func Write(path string, v *VersoriFile) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o600)
}

// FromDir returns the parsed .versori file from `dir`, or (nil, nil) when the
// directory has no .versori. Any other read/parse error is surfaced.
//
// Use this when "no file" is the expected baseline (e.g. context resolution
// outside any synced project) and you want to branch on presence without
// stringly checking os.IsNotExist at every call site.
//
// A `.versori` entry that exists but is not a regular file is also treated as
// absent — the home directory holds the CLI config under `~/.versori/` (a
// directory), and running a project-scoped command from `$HOME` shouldn't fail
// trying to JSON-parse that directory.
func FromDir(dir string) (*VersoriFile, error) {
	if dir == "" {
		return nil, nil
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	path := filepath.Join(absDir, ".versori")

	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}

		return nil, err
	}

	// `.versori` exists but is a directory (or socket, device, etc.) — treat as
	// absent. The home dir's `~/.versori/` is the canonical example.
	if !info.Mode().IsRegular() {
		return nil, nil
	}

	v, err := Read(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}

		return nil, err
	}

	return v, nil
}
