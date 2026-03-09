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

//go:build e2e && !windows

package e2e

import "testing"

// Shared test fixtures reused by context tests and potentially by later tests
// that require an active context.
const (
	testContextName = "test-context"
	testOrg         = "01ARYZ6S41TPTWG9BAXN2R4S4T"

	// A structurally valid JWT with a fake signature.  Flag-based invocations
	// skip the interactive validation path, so any well-formed token works.
	testJWT = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9" +
		".eyJzdWIiOiJ0ZXN0LXVzZXIiLCJpYXQiOjE3MDAwMDAwMDB9" +
		".fake_signature_for_testing_only"
)

// testContextAdd is the first step in the suite.  It creates the context that
// every subsequent step relies on, so if it fails the parent TestCLI stops
// immediately via t.Fatal and later steps are skipped.
func testContextAdd(t *testing.T) {
	_, stderr, code := versori(t,
		"context", "add",
		"--name", testContextName,
		"--organisation", testOrg,
		"--jwt", testJWT,
	)
	if code != 0 {
		t.Fatalf("context add: expected exit 0, got %d\nstderr: %s", code, stderr)
	}

	cfg := readConfig(t)

	if cfg.ActiveContext != testContextName {
		t.Errorf("active_context: got %q, want %q", cfg.ActiveContext, testContextName)
	}

	ctx, ok := cfg.Contexts[testContextName]
	if !ok {
		t.Fatalf("context %q not found in config; present: %v", testContextName, cfg.Contexts)
	}
	if ctx.JWT != testJWT {
		t.Errorf("jwt: got %q, want %q", ctx.JWT, testJWT)
	}
	if ctx.OrganisationId != testOrg {
		t.Errorf("organisation_id: got %q, want %q", ctx.OrganisationId, testOrg)
	}
}

