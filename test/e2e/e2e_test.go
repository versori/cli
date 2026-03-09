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

// Package e2e contains end-to-end tests for the versori CLI.
//
// Tests are intentionally sequential and stateful: each one builds on the
// shared config written by the previous one (e.g. a context must exist before
// any command that talks to the API can run).
//
// The single entry-point is TestCLI, which drives named subtests in an
// explicit order.  Step functions live in numbered files (01_context_test.go,
// 02_…_test.go, …) so the sequence is easy to read on disk, and each subtest
// name carries the same prefix so `go test -v` output reflects the order too.
//
// The suite assumes that the `versori` binary is already compiled and
// available on $PATH.  Run `make cli` (or `go build -o bin/versori .`) and
// add `bin/` to your $PATH before executing these tests.
//
// Inspired by the kubectl e2e suite:
// https://github.com/kubernetes/kubernetes/blob/master/test/e2e/kubectl/kubectl.go
package e2e

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// configPath is the path to the versori config file shared by every step in
// this suite.  It is set once in TestMain and valid for the lifetime of the
// test binary.
var configPath string

// TestMain creates a shared temporary directory that acts as the versori
// config store for all tests, sets VERSORI_CONFIG so that every subprocess
// inherits it automatically, then hands control back to the test runner.
// The directory is removed after all tests finish.
func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "versori-e2e-*")
	if err != nil {
		panic("e2e: failed to create temp dir: " + err.Error())
	}

	configPath = tmpDir + "/config.yaml"

	if err := os.Setenv("VERSORI_CONFIG", configPath); err != nil {
		panic("e2e: failed to set VERSORI_CONFIG: " + err.Error())
	}

	code := m.Run()

	_ = os.RemoveAll(tmpDir)
	os.Exit(code)
}

// versori runs the `versori` binary (assumed to be on $PATH) with the
// supplied arguments and returns trimmed stdout, trimmed stderr, and the
// process exit code.
//
// Because VERSORI_CONFIG is already set in the current process environment
// (by TestMain), every child process inherits it without any extra wiring.
//
// The helper never calls t.Fatal on a non-zero exit code so that callers can
// assert on failure cases as well as success cases.
func versori(t *testing.T, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()

	cmd := exec.Command("versori", args...)

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	stdout = strings.TrimSpace(outBuf.String())
	stderr = strings.TrimSpace(errBuf.String())

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return stdout, stderr, exitErr.ExitCode()
		}
		// Something went wrong before the process even started (e.g. binary
		// not found).  That is always a hard test failure.
		t.Fatalf("versori: failed to run command %v: %v", args, err)
	}

	return stdout, stderr, 0
}

// TestCLI is the single top-level test that drives the entire e2e suite.
// Subtests run in the order listed here; if a subtest calls t.Fatal the
// parent stops immediately so later steps (which may depend on earlier ones)
// are never attempted with a broken state.
//
// Add new steps by creating a new numbered file and appending a t.Run call
// below.
func TestCLI(t *testing.T) {
	t.Run("01/ContextAdd", testContextAdd)
}

// ---------------------------------------------------------------------------
// Shared config helpers (used by step functions across all numbered files)
// ---------------------------------------------------------------------------

// configFile mirrors the on-disk config structure so tests can inspect what
// the CLI wrote without importing internal packages.
type configFile struct {
	ActiveContext string                 `yaml:"active_context"`
	Contexts      map[string]contextFile `yaml:"contexts"`
}

type contextFile struct {
	Name           string `yaml:"name"`
	OrganisationId string `yaml:"organisation_id"`
	JWT            string `yaml:"jwt"`
}

// readConfig reads and unmarshals the shared config file written by the CLI.
func readConfig(t *testing.T) configFile {
	t.Helper()

	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("readConfig: %v", err)
	}

	var cfg configFile
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("readConfig: failed to unmarshal: %v", err)
	}

	return cfg
}

