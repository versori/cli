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
	"net/http"
	"strings"

	"github.com/spf13/cobra"

	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/utils"
)

type get struct {
	configFactory *config.ConfigFactory
	scope         kvScope
	key           string
	rawValues     bool
}

// NewGet builds `kv get` — fetch a single entry by key. Read-only and safe to run for diagnosis.
func NewGet(c *config.ConfigFactory) *cobra.Command {
	g := &get{configFactory: c}

	cmd := &cobra.Command{
		Use:   "get (--store <id> | --scope <scope> ...) --key <a/b/c>",
		Short: "Get a single KV entry by key",
		Long: `Fetch one entry by its key. In raw mode (--store) the --key is the literal full key. In
friendly mode (--scope) the --key is relative to the scope and the CLI prepends the scope prefix.

Values written by workflows are JSON-encoded strings and are unwrapped one level by default; pass
--raw-values to show exactly what is stored.`,
		Run: g.Run,
	}

	f := cmd.Flags()
	g.scope.addFlags(f)
	f.StringVar(&g.key, "key", "", "Key to fetch (slash-delimited, e.g. cursor/orders).")
	f.BoolVar(&g.rawValues, "raw-values", false, "Show the value exactly as stored (no JSON-string unwrapping).")

	_ = cmd.MarkFlagRequired("key")

	return cmd
}

func (g *get) Run(_ *cobra.Command, _ []string) {
	storeID, scopePrefix, friendly := g.scope.resolveStore(g.configFactory)

	keyParts := splitKey(g.key)
	if len(keyParts) == 0 {
		utils.NewExitError().WithMessage("--key must not be empty").Done()
	}

	fullParts := append(append([]string{}, scopePrefix...), keyParts...)
	fullKey := strings.Join(fullParts, "/")

	resp := rawGetResponse{}
	err := g.configFactory.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(&resp).
		WithPath(kvPath(storeID, fullKey)).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to get KV entry (key may not exist)").WithReason(err).Done()
	}

	displayKey := g.key
	if !friendly {
		displayKey = fullKey
	}

	renderEntry(g.configFactory, displayEntry{
		Key:          displayKey,
		Value:        decodeValue(resp.Value, g.rawValues),
		Versionstamp: resp.Versionstamp,
		CreatedAt:    resp.CreatedAt,
		ExpiresAt:    resp.ExpiresAt,
	})
}
