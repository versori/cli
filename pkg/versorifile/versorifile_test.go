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

package versorifile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFromDir(t *testing.T) {
	t.Run("missing file returns nil, nil", func(t *testing.T) {
		dir := t.TempDir()

		got, err := FromDir(dir)
		if err != nil {
			t.Fatalf("FromDir: unexpected error: %v", err)
		}
		if got != nil {
			t.Fatalf("FromDir: expected nil, got %+v", got)
		}
	})

	t.Run("empty dir returns nil, nil", func(t *testing.T) {
		got, err := FromDir("")
		if err != nil {
			t.Fatalf("FromDir(\"\"): unexpected error: %v", err)
		}
		if got != nil {
			t.Fatalf("FromDir(\"\"): expected nil, got %+v", got)
		}
	})

	t.Run("present file is parsed", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".versori")
		if err := os.WriteFile(path, []byte(`{"project_id":"01ABC","context":"dev"}`), 0o600); err != nil {
			t.Fatalf("seed: %v", err)
		}

		got, err := FromDir(dir)
		if err != nil {
			t.Fatalf("FromDir: %v", err)
		}
		if got == nil {
			t.Fatal("FromDir: expected populated file, got nil")
		}
		if got.ProjectId != "01ABC" || got.Context != "dev" {
			t.Fatalf("FromDir: parsed wrong fields: %+v", got)
		}
	})

	t.Run("malformed file surfaces error", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, ".versori")
		if err := os.WriteFile(path, []byte(`{not json`), 0o600); err != nil {
			t.Fatalf("seed: %v", err)
		}

		got, err := FromDir(dir)
		if err == nil {
			t.Fatalf("FromDir: expected error on malformed JSON, got %+v", got)
		}
	})

	// The home directory holds the CLI config under `~/.versori/`, so running
	// a project-scoped command from $HOME used to die trying to JSON-parse the
	// config directory. FromDir now treats any non-regular `.versori` entry
	// (directory, socket, …) as absent.
	t.Run(".versori as a directory is treated as absent", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.Mkdir(filepath.Join(dir, ".versori"), 0o755); err != nil {
			t.Fatalf("seed dir: %v", err)
		}

		got, err := FromDir(dir)
		if err != nil {
			t.Fatalf("FromDir: unexpected error when .versori is a directory: %v", err)
		}
		if got != nil {
			t.Fatalf("FromDir: expected nil for directory .versori, got %+v", got)
		}
	})
}

func TestWriteRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".versori")

	in := &VersoriFile{ProjectId: "01XYZ", Context: "prod"}
	if err := Write(path, in); err != nil {
		t.Fatalf("Write: %v", err)
	}

	out, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if *out != *in {
		t.Fatalf("round-trip mismatch: got %+v, want %+v", out, in)
	}
}
