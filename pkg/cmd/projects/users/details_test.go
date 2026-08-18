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

package users

import (
	"testing"

	v1 "github.com/versori/cli/pkg/api/v1"
	"github.com/versori/cli/pkg/ulid"
)

func TestResolveDetailConnectionsFallsBackWhenActivationOmitsThem(t *testing.T) {
	fallback := []v1.Connection{{
		ID:   ulid.MustDefault(),
		Name: "embedded-mirakl",
	}}

	got := resolveDetailConnections(nil, fallback)
	if len(got) != 1 || got[0].Name != "embedded-mirakl" {
		t.Fatalf("expected fallback connections when GET activation omitted them, got %#v", got)
	}
}

func TestResolveDetailConnectionsFallsBackWhenActivationReturnsEmpty(t *testing.T) {
	fallback := []v1.Connection{{Name: "embedded-mirakl"}}

	got := resolveDetailConnections([]v1.Connection{}, fallback)
	if len(got) != 1 || got[0].Name != "embedded-mirakl" {
		t.Fatalf("expected fallback connections when GET activation returned none, got %#v", got)
	}
}

func TestResolveDetailConnectionsPrefersActivationBindings(t *testing.T) {
	bound := []v1.Connection{{Name: "bound-on-activation"}}
	fallback := []v1.Connection{{Name: "other-embedded"}}

	got := resolveDetailConnections(bound, fallback)
	if len(got) != 1 || got[0].Name != "bound-on-activation" {
		t.Fatalf("expected activation-bound connections to win, got %#v", got)
	}
}

func TestParseActivationDetailsKeepsConnectionIdentityWhenCredentialsAreNoisy(t *testing.T) {
	id := ulid.MustDefault()
	userID := ulid.MustDefault()
	raw := []byte(`{
		"id": "` + id.String() + `",
		"user": {"id": "` + userID.String() + `", "externalId": "acme", "displayName": "Acme"},
		"dynamicVariables": {"channel": "eu"},
		"connections": [{
			"id": "` + id.String() + `",
			"name": "Acme Mirakl",
			"systemId": "` + userID.String() + `",
			"connectionTemplateId": "` + id.String() + `",
			"baseUrl": "https://acme.mirakl.net",
			"credentials": [{"id": "", "authSchemeConfig": {"type": "jwt-bearer", "unknown": true}}]
		}]
	}`)

	got, err := parseActivationDetails(raw)
	if err != nil {
		t.Fatalf("parseActivationDetails: %v", err)
	}
	if got.User.ExternalID != "acme" {
		t.Fatalf("user external id: got %q", got.User.ExternalID)
	}
	if len(got.Connections) != 1 || got.Connections[0].Name != "Acme Mirakl" {
		t.Fatalf("expected connection identity to survive noisy credentials, got %#v", got.Connections)
	}
	if got.Connections[0].ID.String() != id.String() {
		t.Fatalf("connection id: got %q want %q", got.Connections[0].ID.String(), id.String())
	}
}

func TestParseActivationDetailsEmptyConnectionsWhenOmitted(t *testing.T) {
	id := ulid.MustDefault()
	raw := []byte(`{"id": "` + id.String() + `", "user": {"externalId": "acme"}}`)

	got, err := parseActivationDetails(raw)
	if err != nil {
		t.Fatalf("parseActivationDetails: %v", err)
	}
	if len(got.Connections) != 0 {
		t.Fatalf("expected no connections when the field is omitted, got %#v", got.Connections)
	}
}
