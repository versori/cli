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

package projects

import (
	"os"
	"path/filepath"
	"testing"

	v1 "github.com/versori/cli/pkg/api/v1"
)

func TestUpdateFileDryRunAction(t *testing.T) {
	tests := []struct {
		name    string
		local   *string
		remote  string
		want    fileAction
		wantStr string
	}{
		{name: "missing locally", local: nil, remote: "hello", want: actionCreate, wantStr: "create"},
		{name: "identical contents", local: ptr("hello"), remote: "hello", want: actionUnchanged, wantStr: "unchanged"},
		{name: "differing contents", local: ptr("hello"), remote: "goodbye", want: actionUpdate, wantStr: "update"},
		{name: "empty local vs content", local: ptr(""), remote: "hello", want: actionUpdate, wantStr: "update"},
		{name: "both empty", local: ptr(""), remote: "", want: actionUnchanged, wantStr: "unchanged"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, dryRun := range []bool{true, false} {
				dir := t.TempDir()

				if tt.local != nil {
					if err := os.WriteFile(filepath.Join(dir, "a.ts"), []byte(*tt.local), 0o600); err != nil {
						t.Fatalf("failed to seed local file: %v", err)
					}
				}

				existing := map[string]struct{}{"a.ts": {}}

				rel, got := updateFile(v1.File{Filename: "a.ts", Content: tt.remote}, dir, existing, dryRun)
				if got != tt.want {
					t.Fatalf("dryRun=%v: updateFile() action = %v, want %v (%s)", dryRun, got, tt.want, tt.wantStr)
				}

				if rel != "a.ts" {
					t.Fatalf("dryRun=%v: updateFile() rel = %q, want %q", dryRun, rel, "a.ts")
				}

				if _, stillThere := existing["a.ts"]; stillThere {
					t.Fatalf("dryRun=%v: synced file was left in the deletion set", dryRun)
				}

				// A dry-run must never touch the working tree.
				if dryRun && tt.local == nil {
					if _, err := os.Stat(filepath.Join(dir, "a.ts")); !os.IsNotExist(err) {
						t.Fatal("dry-run created a file on disk")
					}
				}

				if !dryRun {
					content, err := os.ReadFile(filepath.Join(dir, "a.ts"))
					if err != nil {
						t.Fatalf("expected file to exist after real sync: %v", err)
					}

					if string(content) != tt.remote {
						t.Fatalf("after real sync content = %q, want %q", content, tt.remote)
					}
				}
			}
		})
	}
}

// Directories are not files, so a directory holding synced files must never be
// reported as affected by a sync.
func TestGetExistingFilesSeparatesDirs(t *testing.T) {
	dir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(dir, "src", "nested"), 0o755); err != nil {
		t.Fatalf("failed to create dirs: %v", err)
	}

	for _, rel := range []string{"root.ts", filepath.Join("src", "a.ts"), filepath.Join("src", "nested", "b.ts")} {
		if err := os.WriteFile(filepath.Join(dir, rel), []byte("x"), 0o600); err != nil {
			t.Fatalf("failed to seed %s: %v", rel, err)
		}
	}

	existing, dirs := getExistingFiles(dir)

	if len(existing) != 3 {
		t.Fatalf("existing = %v, want 3 files", existing)
	}

	for rel := range existing {
		info, err := os.Stat(filepath.Join(dir, rel))
		if err != nil {
			t.Fatalf("failed to stat %s: %v", rel, err)
		}

		if info.IsDir() {
			t.Fatalf("directory %q was included in the affected-files set", rel)
		}
	}

	if len(dirs) != 2 {
		t.Fatalf("dirs = %v, want 2 directories", dirs)
	}
}

func TestPruneEmptyDirsKeepsNonEmpty(t *testing.T) {
	dir := t.TempDir()

	if err := os.MkdirAll(filepath.Join(dir, "empty", "deeper"), 0o755); err != nil {
		t.Fatalf("failed to create dirs: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(dir, "kept"), 0o755); err != nil {
		t.Fatalf("failed to create dirs: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "kept", "a.ts"), []byte("x"), 0o600); err != nil {
		t.Fatalf("failed to seed file: %v", err)
	}

	pruneEmptyDirs(dir, []string{"empty", filepath.Join("empty", "deeper"), "kept"})

	if _, err := os.Stat(filepath.Join(dir, "empty")); !os.IsNotExist(err) {
		t.Fatal("empty parent dir was not pruned")
	}

	if _, err := os.Stat(filepath.Join(dir, "kept", "a.ts")); err != nil {
		t.Fatalf("non-empty dir was pruned: %v", err)
	}
}

func ptr(s string) *string {
	return &s
}
