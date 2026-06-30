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

package kv

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestFingerprintExternalUserID(t *testing.T) {
	// Parity with @versori/run's fingerprintExternalUserId (sha256 hex). The expected values are
	// the canonical sha256 digests of the inputs, so this guards against the hash drifting.
	cases := map[string]string{
		"":      "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		"hello": "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
	}

	for in, want := range cases {
		if got := fingerprintExternalUserID(in); got != want {
			t.Errorf("fingerprintExternalUserID(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestResolveScopePrefix(t *testing.T) {
	const (
		org   = "01ORG"
		proj  = "01PROJ"
		env   = "production"
		exec  = "01EXEC"
		act   = "01ACT"
		extID = "merchant-42"
	)

	userHash := fingerprintExternalUserID(extID)

	tests := []struct {
		name         string
		scope        string
		external     string
		execution    string
		activation   string
		wantStore    string
		wantPrefix   []string
		wantErr      bool
	}{
		{
			name:       "organization",
			scope:      "organization",
			wantStore:  "ORG_" + org,
			wantPrefix: []string{org},
		},
		{
			name:       "org alias",
			scope:      "org",
			wantStore:  "ORG_" + org,
			wantPrefix: []string{org},
		},
		{
			name:       "workspace",
			scope:      "workspace",
			wantStore:  "PROJECT_" + proj,
			wantPrefix: []string{org, proj, env},
		},
		{
			name:       "project",
			scope:      "project",
			wantStore:  "PROJECT_" + proj,
			wantPrefix: []string{org, proj, env},
		},
		{
			name:       "project with activation",
			scope:      "project",
			activation: act,
			wantStore:  "PROJECT_" + proj,
			wantPrefix: []string{org, proj, env, act},
		},
		{
			name:       "user",
			scope:      "user",
			external:   extID,
			wantStore:  "PROJECT_" + proj,
			wantPrefix: []string{org, proj, env, userHash},
		},
		{
			name:    "user without external id",
			scope:   "user",
			wantErr: true,
		},
		{
			name:       "execution",
			scope:      "execution",
			execution:  exec,
			wantStore:  "EXECUTION_" + proj,
			wantPrefix: []string{org, proj, env, exec},
		},
		{
			name:       "execution with activation",
			scope:      "execution",
			execution:  exec,
			activation: act,
			wantStore:  "EXECUTION_" + proj,
			wantPrefix: []string{org, proj, env, exec, act},
		},
		{
			name:      "execution without id",
			scope:     "execution",
			wantErr:   true,
		},
		{
			name:    "unknown scope",
			scope:   "bogus",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := resolveScopePrefix(tt.scope, org, proj, env, tt.external, tt.execution, tt.activation)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got none (result %+v)", res)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if res.storeName != tt.wantStore {
				t.Errorf("storeName = %q, want %q", res.storeName, tt.wantStore)
			}

			if !reflect.DeepEqual(res.prefix, tt.wantPrefix) {
				t.Errorf("prefix = %v, want %v", res.prefix, tt.wantPrefix)
			}
		})
	}
}

func TestEncodeValueForSetMatchesSDKEncoding(t *testing.T) {
	// The SDK JSON.stringify()s values before storing. encodeValueForSet must produce the same
	// canonical JSON text so workflow reads round-trip.
	tests := []struct {
		in   string
		want string
	}{
		{in: `{"a":1}`, want: `{"a":1}`},
		{in: `[1,2,3]`, want: `[1,2,3]`},
		{in: `42`, want: `42`},
		{in: `true`, want: `true`},
		{in: `"already a string"`, want: `"already a string"`},
		{in: `hello`, want: `"hello"`}, // not valid JSON -> stored as a string
	}

	for _, tt := range tests {
		if got := encodeValueForSet(tt.in); got != tt.want {
			t.Errorf("encodeValueForSet(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestValueRoundTrip(t *testing.T) {
	// Full round trip: encode as the CLI would send it, simulate the platform storing that string
	// and returning it as JSON, then decode for display. The decoded value must equal the logical
	// input the user intended.
	tests := []struct {
		name string
		in   string
		want interface{}
	}{
		{name: "object", in: `{"a":1}`, want: map[string]interface{}{"a": float64(1)}},
		{name: "array", in: `[1,2]`, want: []interface{}{float64(1), float64(2)}},
		{name: "number", in: `42`, want: float64(42)},
		{name: "bool", in: `true`, want: true},
		{name: "string", in: `hello`, want: "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := encodeValueForSet(tt.in)

			// The platform stores `encoded` as a string value; a read returns its JSON encoding.
			stored, err := json.Marshal(encoded)
			if err != nil {
				t.Fatalf("marshal stored value: %v", err)
			}

			got := decodeValue(stored, false)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("round trip of %q = %#v, want %#v", tt.in, got, tt.want)
			}
		})
	}
}

func TestDecodeValueRawValues(t *testing.T) {
	// With rawValues, a JSON-string value is shown literally rather than unwrapped.
	stored := json.RawMessage(`"{\"a\":1}"`)

	if got := decodeValue(stored, true); got != `{"a":1}` {
		t.Errorf("decodeValue(raw=true) = %#v, want the literal JSON string", got)
	}

	unwrapped := decodeValue(stored, false)
	if !reflect.DeepEqual(unwrapped, map[string]interface{}{"a": float64(1)}) {
		t.Errorf("decodeValue(raw=false) = %#v, want unwrapped object", unwrapped)
	}
}

func TestStripKey(t *testing.T) {
	key := []string{"01ORG", "01PROJ", "production", "users", "123"}

	if got := stripKey(key, 3); got != "users/123" {
		t.Errorf("stripKey(.., 3) = %q, want %q", got, "users/123")
	}

	if got := stripKey(key, 0); got != "01ORG/01PROJ/production/users/123" {
		t.Errorf("stripKey(.., 0) = %q, want full key", got)
	}

	// stripN larger than the key length must not panic and yields an empty string.
	if got := stripKey([]string{"a"}, 5); got != "" {
		t.Errorf("stripKey overflow = %q, want empty", got)
	}
}

func TestParseFilterTime(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "2026-06-01", want: "2026-06-01T00:00:00Z"},
		{in: "2026-06-01T12:30:00Z", want: "2026-06-01T12:30:00Z"},
		{in: "2026-06-01T12:30:00", want: "2026-06-01T12:30:00Z"},
		{in: "2026-06-01T12:30:00+01:00", want: "2026-06-01T11:30:00Z"}, // normalised to UTC
	}

	for _, tt := range tests {
		if got := parseFilterTime(tt.in, "--created-after"); got != tt.want {
			t.Errorf("parseFilterTime(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestParseMetadataPairs(t *testing.T) {
	got := parseMetadataPairs([]string{
		"status=failed",
		"attempts=3",
		"active=true",
		"shape={\"nested\":1}",
	})

	want := map[string]interface{}{
		"status":  "failed",
		"attempts": float64(3),
		"active":   true,
		"shape":    map[string]interface{}{"nested": float64(1)},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseMetadataPairs = %#v, want %#v", got, want)
	}
}

func TestFilterFlagsBuild(t *testing.T) {
	// No flags set -> nil filter (so the request omits the filter entirely).
	empty := &filterFlags{}
	if empty.build() != nil {
		t.Errorf("build() with no flags = non-nil, want nil")
	}

	ff := &filterFlags{createdAfter: "2026-06-01", metadata: []string{"k=v"}}
	got := ff.build()
	if got == nil {
		t.Fatal("build() with flags = nil, want non-nil")
	}

	if got.CreatedAfter != "2026-06-01T00:00:00Z" {
		t.Errorf("CreatedAfter = %q, want normalised RFC3339", got.CreatedAfter)
	}

	if got.CreatedBefore != "" {
		t.Errorf("CreatedBefore = %q, want empty (unset)", got.CreatedBefore)
	}

	if !reflect.DeepEqual(got.Metadata, map[string]interface{}{"k": "v"}) {
		t.Errorf("Metadata = %#v, want {k:v}", got.Metadata)
	}
}

func TestEnsureWipePrefixSafe(t *testing.T) {
	tests := []struct {
		name    string
		prefix  []string
		wantErr bool
	}{
		{name: "nil prefix", prefix: nil, wantErr: true},
		{name: "empty slice", prefix: []string{}, wantErr: true},
		{name: "single empty segment (--prefix '')", prefix: []string{""}, wantErr: true},
		{name: "trailing empty segment", prefix: []string{"a", "b", ""}, wantErr: true},
		{name: "leading empty segment", prefix: []string{"", "a"}, wantErr: true},
		{name: "valid single segment", prefix: []string{"users"}, wantErr: false},
		{name: "valid multi segment", prefix: []string{"01ORG", "01PROJ", "production"}, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ensureWipePrefixSafe(tt.prefix)
			if tt.wantErr && err == nil {
				t.Errorf("ensureWipePrefixSafe(%v) = nil, want error", tt.prefix)
			}

			if !tt.wantErr && err != nil {
				t.Errorf("ensureWipePrefixSafe(%v) = %v, want nil", tt.prefix, err)
			}
		})
	}
}

func TestSplitKey(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{in: "a/b/c", want: []string{"a", "b", "c"}},
		{in: "/a/b/", want: []string{"a", "b"}},
		{in: "single", want: []string{"single"}},
		{in: "", want: nil},
		{in: "/", want: nil},
	}

	for _, tt := range tests {
		if got := splitKey(tt.in); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("splitKey(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}
