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

func TestResolveDeleteUserIDAcceptsAULID(t *testing.T) {
	id := ulid.MustDefault()
	got, err := resolveDeleteUserID(id.String(), nil)
	if err != nil {
		t.Fatalf("resolveDeleteUserID: %v", err)
	}
	if got != id.String() {
		t.Fatalf("got %q, want the ULID unchanged", got)
	}
}

func TestResolveDeleteUserIDMapsExternalID(t *testing.T) {
	id := ulid.MustDefault()
	got, err := resolveDeleteUserID("george", []v1.EndUser{{
		ID:         id,
		ExternalID: "george",
	}})
	if err != nil {
		t.Fatalf("resolveDeleteUserID: %v", err)
	}
	if got != id.String() {
		t.Fatalf("got %q, want %q", got, id.String())
	}
}

func TestResolveDeleteUserIDRejectsUnknownExternalID(t *testing.T) {
	_, err := resolveDeleteUserID("missing", nil)
	if err == nil {
		t.Fatal("expected an error for an unknown external id")
	}
}
