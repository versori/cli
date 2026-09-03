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

package selfupdate

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPickVSIX(t *testing.T) {
	base := CompatTable{
		ExtensionID: "versori.versori-vscode",
		Releases: []CompatRelease{
			{Vsix: "0.1.0", MinCLI: "0.1.0"},
		},
	}
	twoRows := CompatTable{
		ExtensionID: "versori.versori-vscode",
		Releases: []CompatRelease{
			{Vsix: "0.1.0", MinCLI: "0.1.0"},
			{Vsix: "0.4.0", MinCLI: "0.3.0"},
		},
	}

	cases := []struct {
		name       string
		cliVersion string
		table      CompatTable
		want       string
		ok         bool
	}{
		{
			name:       "cli older than first min_cli",
			cliVersion: "0.0.9",
			table:      base,
			ok:         false,
		},
		{
			name:       "cli equals min_cli",
			cliVersion: "0.1.0",
			table:      base,
			want:       "0.1.0",
			ok:         true,
		},
		{
			name:       "cli newer than min_cli same vsix",
			cliVersion: "0.1.1",
			table:      base,
			want:       "0.1.0",
			ok:         true,
		},
		{
			name:       "newer vsix with higher min_cli left unpicked",
			cliVersion: "0.1.0",
			table:      twoRows,
			want:       "0.1.0",
			ok:         true,
		},
		{
			name:       "cli meets higher min_cli picks newest vsix",
			cliVersion: "0.3.0",
			table:      twoRows,
			want:       "0.4.0",
			ok:         true,
		},
		{
			name:       "empty table",
			cliVersion: "0.1.0",
			table:      CompatTable{},
			ok:         false,
		},
		{
			name:       "leading v on cli and table",
			cliVersion: "v0.1.0",
			table: CompatTable{Releases: []CompatRelease{
				{Vsix: "v0.1.0", MinCLI: "v0.1.0"},
			}},
			want: "v0.1.0",
			ok:   true,
		},
		{
			name:       "invalid row skipped",
			cliVersion: "0.1.0",
			table: CompatTable{Releases: []CompatRelease{
				{Vsix: "not-a-version", MinCLI: "0.0.1"},
				{Vsix: "0.1.0", MinCLI: "0.1.0"},
			}},
			want: "0.1.0",
			ok:   true,
		},
		{
			name:       "invalid cli version",
			cliVersion: "latest",
			table:      base,
			ok:         false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := PickVSIX(tc.cliVersion, tc.table)
			if ok != tc.ok || got != tc.want {
				t.Fatalf("PickVSIX(%q) = %q, %v; want %q, %v", tc.cliVersion, got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestParseCompat(t *testing.T) {
	table, err := ParseCompat([]byte(`{
  "extension_id": "versori.versori-vscode",
  "releases": [
    { "vsix": "0.1.0", "min_cli": "0.1.0" }
  ]
}`))
	if err != nil {
		t.Fatalf("ParseCompat: %v", err)
	}
	if table.ExtensionID != "versori.versori-vscode" {
		t.Fatalf("extension_id=%q", table.ExtensionID)
	}
	if len(table.Releases) != 1 || table.Releases[0].Vsix != "0.1.0" || table.Releases[0].MinCLI != "0.1.0" {
		t.Fatalf("releases=%v", table.Releases)
	}

	if _, err := ParseCompat([]byte(`{not json`)); err == nil {
		t.Fatal("ParseCompat malformed JSON: want error")
	}
}

func TestFetchCompat(t *testing.T) {
	t.Run("404", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		t.Cleanup(srv.Close)

		_, err := FetchCompat(srv.Client(), srv.URL)
		if err == nil {
			t.Fatal("FetchCompat 404: want error")
		}
	})

	t.Run("200 parse error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{not json`))
		}))
		t.Cleanup(srv.Close)

		_, err := FetchCompat(srv.Client(), srv.URL)
		if err == nil {
			t.Fatal("FetchCompat bad body: want error")
		}
	})

	t.Run("200 ok", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"extension_id":"versori.versori-vscode","releases":[{"vsix":"0.1.0","min_cli":"0.1.0"}]}`))
		}))
		t.Cleanup(srv.Close)

		table, err := FetchCompat(srv.Client(), srv.URL)
		if err != nil {
			t.Fatal(err)
		}
		got, ok := PickVSIX("0.1.0", table)
		if !ok || got != "0.1.0" {
			t.Fatalf("got %q, %v", got, ok)
		}
	})
}

func TestTooOldMessage(t *testing.T) {
	msg := TooOldMessage()
	if !strings.Contains(msg, "versori update") {
		t.Fatalf("TooOldMessage missing versori update: %q", msg)
	}
	if !strings.Contains(msg, "curl -fsSL https://raw.githubusercontent.com/versori/cli/main/install.sh | sh") {
		t.Fatalf("TooOldMessage missing curl install: %q", msg)
	}
}
