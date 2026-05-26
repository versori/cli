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

package config

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveContextName(t *testing.T) {
	type want struct {
		name   string
		source contextSource
	}

	tests := []struct {
		name string
		c    ConfigFactory
		want want
	}{
		{
			name: "nothing configured",
			c:    ConfigFactory{Config: &Config{}},
			want: want{name: "", source: contextSourceNone},
		},
		{
			name: "active context only",
			c: ConfigFactory{
				Config: &Config{ActiveContext: "default"},
			},
			want: want{name: "default", source: contextSourceActive},
		},
		{
			name: ".versori override pinned with no active default",
			c: ConfigFactory{
				Config:              &Config{},
				overrideContextName: "project-ctx",
			},
			want: want{name: "project-ctx", source: contextSourceVersoriFile},
		},
		{
			name: ".versori override pinned beats active",
			c: ConfigFactory{
				Config:              &Config{ActiveContext: "default"},
				overrideContextName: "project-ctx",
			},
			want: want{name: "project-ctx", source: contextSourceVersoriFile},
		},
		{
			name: "explicit --context flag beats override and active",
			c: ConfigFactory{
				Config:              &Config{ActiveContext: "default"},
				overrideContextName: "project-ctx",
				contextName:         "flag-ctx",
			},
			want: want{name: "flag-ctx", source: contextSourceFlag},
		},
		{
			name: "explicit --context flag beats active alone",
			c: ConfigFactory{
				Config:      &Config{ActiveContext: "default"},
				contextName: "flag-ctx",
			},
			want: want{name: "flag-ctx", source: contextSourceFlag},
		},
		{
			name: "no override pin falls through to active",
			c: ConfigFactory{
				Config:              &Config{ActiveContext: "default"},
				overrideContextName: "",
			},
			want: want{name: "default", source: contextSourceActive},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotSource := tt.c.resolveContextName()
			if gotName != tt.want.name {
				t.Errorf("name: got %q, want %q", gotName, tt.want.name)
			}
			if gotSource != tt.want.source {
				t.Errorf("source: got %d, want %d", gotSource, tt.want.source)
			}
		})
	}
}

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

// TestMaybeApplyVersoriContextForProject covers the lazy override that
// flags.ProjectId.GetFlagOrDie invokes once it knows the resolved project
// ID. The override fires only when --context is absent AND the resolved
// project ID matches the .versori file's project_id.
func TestMaybeApplyVersoriContextForProject(t *testing.T) {
	type setup struct {
		factory        *ConfigFactory
		writeVersori   bool
		versoriProject string
		versoriContext string
	}

	type wantState struct {
		override     string
		noticeShown  bool
		noticeOnErr  bool
		noticeSubstr string
	}

	cases := []struct {
		name      string
		setup     setup
		flagCtx   string
		projectId string
		want      wantState
	}{
		{
			name: "matching project id applies override (active differs → notice)",
			setup: setup{
				factory:        &ConfigFactory{Config: &Config{ActiveContext: "default"}},
				writeVersori:   true,
				versoriProject: "01PROJ",
				versoriContext: "project-ctx",
			},
			projectId: "01PROJ",
			want: wantState{
				override:     "project-ctx",
				noticeShown:  true,
				noticeOnErr:  true,
				noticeSubstr: `using context "project-ctx"`,
			},
		},
		{
			name: "matching project id and matching active → override set, no notice",
			setup: setup{
				factory:        &ConfigFactory{Config: &Config{ActiveContext: "project-ctx"}},
				writeVersori:   true,
				versoriProject: "01PROJ",
				versoriContext: "project-ctx",
			},
			projectId: "01PROJ",
			want: wantState{
				override:    "project-ctx",
				noticeShown: true,
				noticeOnErr: false,
			},
		},
		{
			name: "mismatched project id keeps active (no override, no notice)",
			setup: setup{
				factory:        &ConfigFactory{Config: &Config{ActiveContext: "default"}},
				writeVersori:   true,
				versoriProject: "01PROJ",
				versoriContext: "project-ctx",
			},
			projectId: "01OTHER",
			want: wantState{
				override:    "",
				noticeShown: false,
				noticeOnErr: false,
			},
		},
		{
			name: "--context flag suppresses override entirely",
			setup: setup{
				factory:        &ConfigFactory{Config: &Config{ActiveContext: "default"}},
				writeVersori:   true,
				versoriProject: "01PROJ",
				versoriContext: "project-ctx",
			},
			flagCtx:   "flag-ctx",
			projectId: "01PROJ",
			want: wantState{
				override:    "",
				noticeShown: false,
				noticeOnErr: false,
			},
		},
		{
			name: "no .versori in dir → no-op",
			setup: setup{
				factory:      &ConfigFactory{Config: &Config{ActiveContext: "default"}},
				writeVersori: false,
			},
			projectId: "01PROJ",
			want: wantState{
				override:    "",
				noticeShown: false,
				noticeOnErr: false,
			},
		},
		{
			name: "empty projectId → no-op",
			setup: setup{
				factory:        &ConfigFactory{Config: &Config{ActiveContext: "default"}},
				writeVersori:   true,
				versoriProject: "01PROJ",
				versoriContext: "project-ctx",
			},
			projectId: "",
			want: wantState{
				override:    "",
				noticeShown: false,
				noticeOnErr: false,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if tc.setup.writeVersori {
				path := filepath.Join(dir, ".versori")
				content := `{"project_id":"` + tc.setup.versoriProject + `","context":"` + tc.setup.versoriContext + `"}`
				if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
					t.Fatalf("seed: %v", err)
				}
			}

			c := tc.setup.factory
			c.contextName = tc.flagCtx

			stderr := captureStderr(t, func() {
				c.maybeApplyVersoriContextForProject(dir, tc.projectId)
			})

			if c.overrideContextName != tc.want.override {
				t.Errorf("overrideContextName: got %q, want %q", c.overrideContextName, tc.want.override)
			}
			if c.versoriNoticeShown != tc.want.noticeShown {
				t.Errorf("versoriNoticeShown: got %v, want %v", c.versoriNoticeShown, tc.want.noticeShown)
			}
			if tc.want.noticeOnErr {
				if !strings.Contains(stderr, tc.want.noticeSubstr) {
					t.Errorf("expected notice on stderr containing %q, got %q", tc.want.noticeSubstr, stderr)
				}
			} else if stderr != "" {
				t.Errorf("expected silent stderr, got %q", stderr)
			}
		})
	}
}

// TestMaybeApplyVersoriContextForProject_OneShotNotice asserts that even
// when the override would re-fire (e.g. NewRequest invoked many times),
// the notice is emitted exactly once.
func TestMaybeApplyVersoriContextForProject_OneShotNotice(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".versori")
	if err := os.WriteFile(path, []byte(`{"project_id":"01PROJ","context":"project-ctx"}`), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	c := &ConfigFactory{Config: &Config{ActiveContext: "default"}}

	stderr := captureStderr(t, func() {
		c.maybeApplyVersoriContextForProject(dir, "01PROJ")
		c.maybeApplyVersoriContextForProject(dir, "01PROJ")
		c.maybeApplyVersoriContextForProject(dir, "01PROJ")
	})

	count := strings.Count(stderr, `using context "project-ctx"`)
	if count != 1 {
		t.Errorf("notice count: got %d, want 1; stderr=%q", count, stderr)
	}
}

// TestMaybeApplyVersoriContextForProject_PackageFuncNoop verifies the
// exported helper is safe to call when no factory has registered yet
// (e.g. unit tests that never go through LoadConfigAndContext).
func TestMaybeApplyVersoriContextForProject_PackageFuncNoop(t *testing.T) {
	prev := defaultFactory
	defaultFactory = nil
	t.Cleanup(func() { defaultFactory = prev })

	// Should not panic.
	MaybeApplyVersoriContextForProject(t.TempDir(), "01PROJ")
}
