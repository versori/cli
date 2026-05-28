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
