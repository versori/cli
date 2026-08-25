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
	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/cmd/flags"
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

func TestCollapseByDir(t *testing.T) {
	tests := []struct {
		name string
		// group is the set being rendered; others are the remaining known
		// files, which decide whether a directory is uniformly affected.
		group  []string
		others []string
		want   []string
	}{
		{
			name:  "whole directory collapses",
			group: []string{"src/a.ts", "src/b.ts", "src/c.ts"},
			want:  []string{"src/ (3 files)"},
		},
		{
			name:   "mixed directory does not collapse",
			group:  []string{"src/a.ts", "src/b.ts"},
			others: []string{"src/keep.ts"},
			want:   []string{"src/a.ts", "src/b.ts"},
		},
		{
			name:  "collapses to shallowest fully owned dir",
			group: []string{"src/a.ts", "src/nested/b.ts", "src/nested/c.ts"},
			want:  []string{"src/ (3 files)"},
		},
		{
			name:   "collapses nested dir when parent is mixed",
			group:  []string{"src/nested/b.ts", "src/nested/c.ts"},
			others: []string{"src/keep.ts"},
			want:   []string{"src/nested/ (2 files)"},
		},
		{
			name:  "single file in dir stays expanded",
			group: []string{"src/only.ts"},
			want:  []string{"src/only.ts"},
		},
		{
			name:  "root files are never collapsed",
			group: []string{"a.ts", "b.ts"},
			want:  []string{"a.ts", "b.ts"},
		},
		{
			name:  "root files listed alongside collapsed dir",
			group: []string{"package.json", "src/a.ts", "src/b.ts"},
			want:  []string{"package.json", "src/ (2 files)"},
		},
		{
			name:  "independent sibling dirs collapse separately",
			group: []string{"a/one.ts", "a/two.ts", "b/one.ts", "b/two.ts"},
			want:  []string{"a/ (2 files)", "b/ (2 files)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			totals := countByDir(tt.group, tt.others)

			var got []string

			for _, e := range collapseByDir(tt.group, totals) {
				if e.count == 0 {
					got = append(got, e.path)

					continue
				}

				got = append(got, e.path+"/ ("+pluralFiles(e.count)+")")
			}

			if len(got) != len(tt.want) {
				t.Fatalf("collapseByDir() = %v, want %v", got, tt.want)
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("collapseByDir() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestPluralFiles(t *testing.T) {
	for _, tt := range []struct {
		n    int
		want string
	}{{0, "0 files"}, {1, "1 file"}, {2, "2 files"}} {
		if got := pluralFiles(tt.n); got != tt.want {
			t.Fatalf("pluralFiles(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

func ptr(s string) *string {
	return &s
}

// A version id must route sync onto the same endpoint `projects files
// --version` reads, never the project's CurrentFiles.
func TestVersionFilesPathMatchesFilesCommand(t *testing.T) {
	got := versionFilesPath("01PROJECT", "01VERSION")
	want := "o/:organisation/projects/01PROJECT/versions/01VERSION/files"

	if got != want {
		t.Fatalf("versionFilesPath() = %q, want %q", got, want)
	}
}

func TestSyncRunConsumesSelectedFileSource(t *testing.T) {
	tests := []struct {
		name         string
		version      string
		wantPath     string
		wantFilename string
	}{
		{
			name:         "omitted version selects CurrentFiles",
			wantPath:     "o/:organisation/projects/01PROJECT",
			wantFilename: "current.ts",
		},
		{
			name:         "version selects version files",
			version:      "01VERSION",
			wantPath:     "o/:organisation/projects/01PROJECT/versions/01VERSION/files",
			wantFilename: "version.ts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			s := &Sync{
				projectId: flags.ProjectId("01PROJECT"),
				directory: dir,
				confirm:   true,
				version:   tt.version,
				noPin:     true,
				loadSource: func(_ *config.ConfigFactory, source syncFileSource) ([]v1.File, error) {
					if source.path != tt.wantPath {
						t.Fatalf("selected path = %q, want %q", source.path, tt.wantPath)
					}

					file := v1.File{Filename: tt.wantFilename, Content: tt.name}
					switch response := source.response.(type) {
					case *v1.Project:
						response.CurrentFiles.Files = []v1.File{file}
					case *v1.Files:
						response.Files = []v1.File{file}
					default:
						t.Fatalf("unexpected response type %T", source.response)
					}

					return source.files(), nil
				},
			}

			s.Run(nil, nil)

			got, err := os.ReadFile(filepath.Join(dir, tt.wantFilename))
			if err != nil {
				t.Fatalf("Run did not consume selected files: %v", err)
			}
			if string(got) != tt.name {
				t.Fatalf("written content = %q, want %q", got, tt.name)
			}
		})
	}
}

func TestNormalizeVersionId(t *testing.T) {
	tests := []struct {
		name   string
		raw    string
		want   string
		wantOK bool
	}{
		{name: "flag omitted means current files", raw: "", want: "", wantOK: true},
		{name: "plain id", raw: "01VERSION", want: "01VERSION", wantOK: true},
		{name: "surrounding whitespace is trimmed", raw: "  01VERSION\n", want: "01VERSION", wantOK: true},
		{name: "whitespace only is an error, not current files", raw: "   ", want: "", wantOK: false},
		{name: "tab only is an error", raw: "\t", want: "", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := normalizeVersionId(tt.raw)
			if got != tt.want || ok != tt.wantOK {
				t.Fatalf("normalizeVersionId(%q) = (%q, %v), want (%q, %v)", tt.raw, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

// The pin rule, stated once: only a real sync without --no-pin re-pins.
func TestShouldWritePin(t *testing.T) {
	tests := []struct {
		name   string
		dryRun bool
		noPin  bool
		want   bool
	}{
		{name: "live sync re-pins", dryRun: false, noPin: false, want: true},
		{name: "live sync with --no-pin does not", dryRun: false, noPin: true, want: false},
		{name: "dry-run never pins", dryRun: true, noPin: false, want: false},
		{name: "dry-run with --no-pin never pins", dryRun: true, noPin: true, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldWritePin(tt.dryRun, tt.noPin); got != tt.want {
				t.Fatalf("shouldWritePin(dryRun=%v, noPin=%v) = %v, want %v", tt.dryRun, tt.noPin, got, tt.want)
			}
		})
	}
}

// --no-pin must not delete a .versori that is already in the target. The
// deletion set is what sync would remove, so .versori must never be in it.
func TestGetExistingFilesNeverOffersVersoriForDeletion(t *testing.T) {
	dir := t.TempDir()

	for _, rel := range []string{".versori", "index.ts"} {
		if err := os.WriteFile(filepath.Join(dir, rel), []byte("x"), 0o600); err != nil {
			t.Fatalf("failed to seed %s: %v", rel, err)
		}
	}

	existing, _ := getExistingFiles(dir)

	if _, found := existing[".versori"]; found {
		t.Fatal(".versori was offered up for deletion")
	}

	if _, found := existing["index.ts"]; !found {
		t.Fatal("a normal project file was missing from the deletion set")
	}
}

// The flags exist, are independent, and default to today's behaviour.
func TestNewSyncRegistersVersionAndNoPin(t *testing.T) {
	cmd := NewSync(nil)

	version := cmd.Flags().Lookup("version")
	if version == nil {
		t.Fatal("--version is not registered on projects sync")
	}

	if version.DefValue != "" {
		t.Fatalf("--version default = %q, want empty (current files)", version.DefValue)
	}

	noPin := cmd.Flags().Lookup("no-pin")
	if noPin == nil {
		t.Fatal("--no-pin is not registered on projects sync")
	}

	if noPin.DefValue != "false" {
		t.Fatalf("--no-pin default = %q, want \"false\"", noPin.DefValue)
	}
}
