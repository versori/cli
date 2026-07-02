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
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/utils"
)

type set struct {
	configFactory *config.ConfigFactory
	scope         kvScope
	key           string
	value         string
	valueFile     string
	expireIn      int64
	ifNotExists   bool
	yes           bool
}

// NewSet builds `kv set` — write a value at a key. MUTATION: changes live workflow state. Only run
// when the user explicitly asks to write a specific key.
func NewSet(c *config.ConfigFactory) *cobra.Command {
	s := &set{configFactory: c}

	cmd := &cobra.Command{
		Use:   "set (--store <id> | --scope <scope> ...) --key <a/b/c> (--value <json> | --value-file <path>)",
		Short: "Set a KV entry (mutation)",
		Long: `Write a value at a key.

WARNING: this mutates live workflow state (cursors, dedupe keys, batch progress). Only run it when
explicitly asked to set a specific key — never as a side-effect of debugging.

The value is JSON-encoded the same way the runtime SDK writes values, so workflow code that reads
the key with ctx.openKv().get() round-trips correctly. A --value that parses as JSON keeps its type
(object/array/number/bool); otherwise it is stored as a string.

Confirms before writing unless --yes is passed; in non-interactive shells --yes is required.`,
		Run: s.Run,
	}

	f := cmd.Flags()
	s.scope.addFlags(f)
	f.StringVar(&s.key, "key", "", "Key to set (slash-delimited).")
	f.StringVar(&s.value, "value", "", "Value to store (JSON when valid, otherwise a string).")
	f.StringVar(&s.valueFile, "value-file", "", "Read the value from a file instead of --value.")
	f.Int64Var(&s.expireIn, "expire-in", 0, "TTL in milliseconds (0 = no expiry).")
	f.BoolVar(&s.ifNotExists, "if-not-exists", false, "Only set if the key does not already exist.")
	f.BoolVarP(&s.yes, "yes", "y", false, "Skip the confirmation prompt.")

	_ = cmd.MarkFlagRequired("key")

	return cmd
}

func (s *set) Run(_ *cobra.Command, _ []string) {
	storeID, scopePrefix, _ := s.scope.resolveStore(s.configFactory)

	keyParts := splitKey(s.key)
	if len(keyParts) == 0 {
		utils.NewExitError().WithMessage("--key must not be empty").Done()
	}

	raw := s.readValue()

	fullParts := append(append([]string{}, scopePrefix...), keyParts...)
	fullKey := strings.Join(fullParts, "/")

	confirmOrAbort(s.yes, fmt.Sprintf("Set key %q in store %s?", fullKey, storeID))

	body := setBody{Value: encodeValueForSet(raw)}
	if s.expireIn > 0 || s.ifNotExists {
		body.Options = &setOptions{ExpireIn: s.expireIn, IfNotExists: s.ifNotExists}
	}

	resp := struct {
		Committed    *bool   `json:"committed"`
		Versionstamp *string `json:"versionstamp"`
	}{}
	err := s.configFactory.
		NewRequest().
		WithMethod(http.MethodPut).
		JSONBody(body).
		Into(&resp).
		WithPath(kvPath(storeID, fullKey)).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to set KV entry").WithReason(err).Done()
	}

	if resp.Committed != nil && !*resp.Committed {
		fmt.Printf("Not committed (key %q already exists and --if-not-exists was set).\n", fullKey)

		return
	}

	if resp.Versionstamp != nil && *resp.Versionstamp != "" {
		fmt.Printf("Set %q (versionstamp %s).\n", fullKey, *resp.Versionstamp)

		return
	}

	fmt.Printf("Set %q.\n", fullKey)
}

// readValue resolves the value source: --value-file wins over --value; exactly one is required.
func (s *set) readValue() string {
	if s.valueFile != "" {
		b, err := os.ReadFile(s.valueFile)
		if err != nil {
			utils.NewExitError().WithMessage("failed to read --value-file").WithReason(err).Done()
		}

		return string(b)
	}

	if s.value == "" {
		utils.NewExitError().WithMessage("provide --value <json> or --value-file <path>").Done()
	}

	return s.value
}
