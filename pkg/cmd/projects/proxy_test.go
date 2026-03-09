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
	"testing"
)

func TestParseKeyValuePairs(t *testing.T) {
	tests := []struct {
		name     string
		pairs    []string
		flagName string
		want     map[string][]string
	}{
		{
			name:     "empty input",
			pairs:    nil,
			flagName: "header",
			want:     map[string][]string{},
		},
		{
			name:     "single pair",
			pairs:    []string{"Content-Type:application/json"},
			flagName: "header",
			want:     map[string][]string{"Content-Type": {"application/json"}},
		},
		{
			name:     "multiple distinct keys",
			pairs:    []string{"Accept:text/html", "Authorization:Bearer token123"},
			flagName: "header",
			want: map[string][]string{
				"Accept":        {"text/html"},
				"Authorization": {"Bearer token123"},
			},
		},
		{
			name:     "repeated key appends values",
			pairs:    []string{"X-Custom:one", "X-Custom:two"},
			flagName: "header",
			want:     map[string][]string{"X-Custom": {"one", "two"}},
		},
		{
			name:     "value containing colon",
			pairs:    []string{"Authorization:Basic dXNlcjpwYXNz"},
			flagName: "header",
			want:     map[string][]string{"Authorization": {"Basic dXNlcjpwYXNz"}},
		},
		{
			name:     "value with multiple colons",
			pairs:    []string{"filter:http://example.com:8080"},
			flagName: "query",
			want:     map[string][]string{"filter": {"http://example.com:8080"}},
		},
		{
			name:     "whitespace trimmed from key and value",
			pairs:    []string{" key : value "},
			flagName: "header",
			want:     map[string][]string{"key": {"value"}},
		},
		{
			name:     "empty value",
			pairs:    []string{"key:"},
			flagName: "query",
			want:     map[string][]string{"key": {""}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseKeyValuePairs(tt.pairs, tt.flagName)

			if len(got) != len(tt.want) {
				t.Fatalf("got %d keys, want %d keys\ngot:  %v\nwant: %v", len(got), len(tt.want), got, tt.want)
			}

			for k, wantVals := range tt.want {
				gotVals, ok := got[k]
				if !ok {
					t.Fatalf("missing key %q in result\ngot: %v", k, got)
				}
				if len(gotVals) != len(wantVals) {
					t.Fatalf("key %q: got %d values, want %d\ngot:  %v\nwant: %v", k, len(gotVals), len(wantVals), gotVals, wantVals)
				}
				for i := range wantVals {
					if gotVals[i] != wantVals[i] {
						t.Errorf("key %q value[%d]: got %q, want %q", k, i, gotVals[i], wantVals[i])
					}
				}
			}
		})
	}
}
