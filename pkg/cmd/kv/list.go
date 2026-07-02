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

	"github.com/spf13/cobra"

	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/utils"
)

type list struct {
	configFactory *config.ConfigFactory
	scope         kvScope
	filter        filterFlags
	prefix        []string
	limit         int
	reverse       bool
	after         string
	before        string
	rawValues     bool
}

// NewList builds `kv list` — enumerate entries under a prefix. Read-only and safe to run for
// diagnosis.
func NewList(c *config.ConfigFactory) *cobra.Command {
	l := &list{configFactory: c}

	cmd := &cobra.Command{
		Use:   "list (--store <id> | --scope <scope> ...) [--prefix <segment>]...",
		Short: "List KV entries under a prefix",
		Long: `List key/value entries in a store, optionally narrowed by a key prefix.

Targeting (pick one):
  --store <id>     raw mode; --prefix segments are the literal key prefix.
  --scope <scope>  friendly mode; the CLI derives the store + scope prefix and strips it from
                   displayed keys (use --full-keys to keep it). Extra --prefix segments narrow
                   within the scope.

Values written by workflows are JSON-encoded strings; by default they are unwrapped one level for
readability. Pass --raw-values to show exactly what is stored. For nested values prefer -o json or
-o yaml, which preserve the full structure (table truncates the value column).

Beyond the prefix, entries can be filtered server-side by creation time (--created-after /
--created-before) and by metadata (--metadata key=value).

Pagination is cursor-based, ordered newest-first by the entry's internal ID (not by key). A page
is capped (max 100). To page through results, pass the previous response's opaque nextCursor to
--after — never a key. An empty nextCursor means you have reached the end; it does not mean more
pages are hidden. Use 'kv count' for an authoritative total under a prefix.`,
		Run: l.Run,
	}

	f := cmd.Flags()
	l.scope.addFlags(f)
	l.filter.addFlags(f)
	f.StringArrayVar(&l.prefix, "prefix", nil, "Key prefix segment (repeatable).")
	f.IntVar(&l.limit, "limit", 0, "Max entries to return (1-100; 0 lets the server default apply).")
	f.BoolVar(&l.reverse, "reverse", false, "Reverse the sort order.")
	f.StringVar(&l.after, "after", "", "Pagination cursor for the next page: pass the previous response's opaque nextCursor verbatim (NOT a key). Empty nextCursor means there are no more entries.")
	f.StringVar(&l.before, "before", "", "Pagination cursor for the previous page: pass an opaque prevCursor from a prior response (NOT a key).")
	f.BoolVar(&l.rawValues, "raw-values", false, "Show values exactly as stored (no JSON-string unwrapping).")

	return cmd
}

func (l *list) Run(_ *cobra.Command, _ []string) {
	storeID, scopePrefix, friendly := l.scope.resolveStore(l.configFactory)

	apiPrefix := append(append([]string{}, scopePrefix...), l.prefix...)

	body := listBody{
		Selector: listSelector{Prefix: apiPrefix, After: l.after, Before: l.before, Filter: l.filter.build()},
		Options:  listOptions{Limit: l.limit, Reverse: l.reverse},
	}

	resp := rawListResponse{}
	err := l.configFactory.
		NewRequest().
		WithMethod(http.MethodPost).
		JSONBody(body).
		Into(&resp).
		WithPath(kvPath(storeID, "list")).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to list KV entries").WithReason(err).Done()
	}

	stripN := l.scope.stripCount(scopePrefix, friendly)

	entries := make([]displayEntry, 0, len(resp.Data))
	for _, e := range resp.Data {
		entries = append(entries, displayEntry{
			Key:          stripKey(e.Key, stripN),
			Value:        decodeValue(e.Value, l.rawValues),
			Versionstamp: e.Versionstamp,
			CreatedAt:    e.CreatedAt,
			ExpiresAt:    e.ExpiresAt,
		})
	}

	renderEntries(l.configFactory, entries, resp.Pagination)
}
