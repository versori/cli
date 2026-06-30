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
	"github.com/spf13/cobra"

	"github.com/versori/cli/pkg/cmd/config"
)

type count struct {
	configFactory *config.ConfigFactory
	scope         kvScope
	prefix        []string
}

// NewCount builds `kv count` — return how many entries match a prefix without fetching them.
// Read-only and safe to run for diagnosis.
func NewCount(c *config.ConfigFactory) *cobra.Command {
	cnt := &count{configFactory: c}

	cmd := &cobra.Command{
		Use:   "count (--store <id> | --scope <scope> ...) [--prefix <segment>]...",
		Short: "Count KV entries matching a prefix",
		Long: `Count the entries matching a key prefix. Unlike list, this returns a total rather than a
capped page, so it is the right way to answer "how many items are under this prefix?".

Targeting works the same as kv list: --store <id> (raw) or --scope <scope> (friendly), with
optional --prefix segments to narrow further.`,
		Run: cnt.Run,
	}

	f := cmd.Flags()
	cnt.scope.addFlags(f)
	f.StringArrayVar(&cnt.prefix, "prefix", nil, "Key prefix segment (repeatable).")

	return cmd
}

func (cnt *count) Run(_ *cobra.Command, _ []string) {
	storeID, scopePrefix, _ := cnt.scope.resolveStore(cnt.configFactory)

	apiPrefix := append(append([]string{}, scopePrefix...), cnt.prefix...)

	n := countEntries(cnt.configFactory, storeID, apiPrefix)

	cnt.configFactory.Print(printableCount{Count: n})
}
