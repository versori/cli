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

package flags

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// captureStderr swaps os.Stderr for a pipe during fn and returns whatever
// fn writes to it. Used to assert the "--project overrides .versori" warning.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	original := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stderr = w

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()
	_ = w.Close()
	os.Stderr = original

	return <-done
}

func writeVersori(t *testing.T, dir, projectID, ctxName string) {
	t.Helper()
	content := `{"project_id":"` + projectID + `","context":"` + ctxName + `"}`
	if err := os.WriteFile(filepath.Join(dir, ".versori"), []byte(content), 0o600); err != nil {
		t.Fatalf("seed .versori: %v", err)
	}
}

func TestGetProjectIDFromDir(t *testing.T) {
	t.Run("no .versori, no flag returns empty", func(t *testing.T) {
		var p ProjectId
		dir := t.TempDir()
		if got := p.GetProjectIDFromDir(dir); got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})

	t.Run("no .versori, with flag returns flag value", func(t *testing.T) {
		p := ProjectId("01FLAGID")
		dir := t.TempDir()
		if got := p.GetProjectIDFromDir(dir); got != "01FLAGID" {
			t.Errorf("got %q, want %q", got, "01FLAGID")
		}
	})

	t.Run(".versori present, no flag returns file value", func(t *testing.T) {
		var p ProjectId
		dir := t.TempDir()
		writeVersori(t, dir, "01FILEID", "dev")

		if got := p.GetProjectIDFromDir(dir); got != "01FILEID" {
			t.Errorf("got %q, want %q", got, "01FILEID")
		}
	})

	t.Run(".versori present, matching flag returns flag value (no warning)", func(t *testing.T) {
		p := ProjectId("01SAME")
		dir := t.TempDir()
		writeVersori(t, dir, "01SAME", "dev")

		var got string
		stderr := captureStderr(t, func() {
			got = p.GetProjectIDFromDir(dir)
		})

		if got != "01SAME" {
			t.Errorf("got %q, want %q", got, "01SAME")
		}
		if stderr != "" {
			t.Errorf("expected no warning, got %q", stderr)
		}
	})

	t.Run(".versori present, differing flag wins with warning", func(t *testing.T) {
		p := ProjectId("01FLAG")
		dir := t.TempDir()
		writeVersori(t, dir, "01FILE", "dev")

		var got string
		stderr := captureStderr(t, func() {
			got = p.GetProjectIDFromDir(dir)
		})

		if got != "01FLAG" {
			t.Errorf("got %q, want %q", got, "01FLAG")
		}
		if !strings.Contains(stderr, `--project "01FLAG" overrides .versori project "01FILE"`) {
			t.Errorf("expected divergence warning on stderr, got %q", stderr)
		}
	})
}

func TestCrossProjectSummary(t *testing.T) {
	cases := []struct {
		name   string
		action string
		want   []string
	}{
		{
			name:   "sync mentions overwrite and re-pin",
			action: "sync",
			want:   []string{"OVERWRITE", "01FLAG", "01FILE", "/dir", "rewrite .versori"},
		},
		{
			name:   "deploy mentions DEPLOYED and target project",
			action: "deploy",
			want:   []string{"DEPLOYED", "01FLAG", "01FILE", "/dir"},
		},
		{
			name:   "save mentions SAVED and target project",
			action: "save",
			want:   []string{"SAVED", "01FLAG", "01FILE", "/dir"},
		},
		{
			name:   "unknown action falls back to a generic summary",
			action: "frobnicate",
			want:   []string{"frobnicate", "01FLAG", "01FILE", "/dir"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := crossProjectSummary(tc.action, "01FLAG", "01FILE", "/dir")
			for _, want := range tc.want {
				if !strings.Contains(got, want) {
					t.Errorf("summary %q missing %q", got, want)
				}
			}
		})
	}
}

func TestRequireCrossProjectConfirm(t *testing.T) {
	t.Run("no .versori is a no-op", func(t *testing.T) {
		dir := t.TempDir()
		stderr := captureStderr(t, func() {
			requireCrossProjectConfirm(dir, "deploy", "01FLAG", false)
		})
		if stderr != "" {
			t.Errorf("expected no prompt, got %q", stderr)
		}
	})

	t.Run("matching .versori is a no-op", func(t *testing.T) {
		dir := t.TempDir()
		writeVersori(t, dir, "01SAME", "dev")

		stderr := captureStderr(t, func() {
			requireCrossProjectConfirm(dir, "deploy", "01SAME", false)
		})
		if stderr != "" {
			t.Errorf("expected no prompt when projects match, got %q", stderr)
		}
	})

	t.Run("autoConfirm bypasses the prompt on mismatch", func(t *testing.T) {
		dir := t.TempDir()
		writeVersori(t, dir, "01FILE", "dev")

		stderr := captureStderr(t, func() {
			requireCrossProjectConfirm(dir, "deploy", "01FLAG", true)
		})
		if stderr != "" {
			t.Errorf("expected no prompt under autoConfirm, got %q", stderr)
		}
	})

	t.Run("CONFIRM on stdin proceeds without exiting", func(t *testing.T) {
		dir := t.TempDir()
		writeVersori(t, dir, "01FILE", "dev")

		withStdin(t, "CONFIRM\n", func() {
			stderr := captureStderr(t, func() {
				requireCrossProjectConfirm(dir, "deploy", "01FLAG", false)
			})
			if !strings.Contains(stderr, "CROSS-PROJECT DEPLOY") {
				t.Errorf("expected cross-project banner on stderr, got %q", stderr)
			}
			if !strings.Contains(stderr, "01FLAG") || !strings.Contains(stderr, "01FILE") {
				t.Errorf("expected both project ids in summary, got %q", stderr)
			}
		})
	})

	t.Run("lowercase confirm on stdin also proceeds", func(t *testing.T) {
		dir := t.TempDir()
		writeVersori(t, dir, "01FILE", "dev")

		withStdin(t, "confirm\n", func() {
			_ = captureStderr(t, func() {
				requireCrossProjectConfirm(dir, "sync", "01FLAG", false)
			})
		})
	})
}

// withStdin swaps os.Stdin for a pipe pre-loaded with input during fn so
// requireCrossProjectConfirm can read its typed-CONFIRM line without an
// interactive terminal.
func withStdin(t *testing.T, input string, fn func()) {
	t.Helper()

	original := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}

	if _, err := w.WriteString(input); err != nil {
		t.Fatalf("write stdin: %v", err)
	}
	_ = w.Close()

	os.Stdin = r
	defer func() {
		os.Stdin = original
		_ = r.Close()
	}()

	fn()
}
