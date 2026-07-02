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
	"strings"

	"github.com/spf13/cobra"

	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/utils"
)

type wipe struct {
	configFactory *config.ConfigFactory
	scope         kvScope
	prefix        []string
	confirm       bool
}

// NewWipe builds `kv wipe` — cascade-delete every entry under a prefix. MUTATION (bulk): the most
// destructive KV operation. Only run when the user explicitly asks to wipe a prefix/scope.
func NewWipe(c *config.ConfigFactory) *cobra.Command {
	w := &wipe{configFactory: c}

	cmd := &cobra.Command{
		Use:   "wipe (--store <id> --prefix <segment>... | --scope <scope> ...) --confirm",
		Short: "Delete every KV entry under a prefix (bulk mutation)",
		Long: `Cascade-delete all entries under a key prefix.

WARNING: this is the most destructive KV operation and mutates live workflow state in bulk. Only
run it when explicitly asked to wipe a specific prefix or scope — never as a side-effect of
debugging or to "clean up".

An empty prefix is refused (that would target the entire store). The command always counts the
matching entries first and prints what would be deleted; it only proceeds when --confirm is passed,
so running it without --confirm is a safe dry-run.`,
		Run: w.Run,
	}

	f := cmd.Flags()
	w.scope.addFlags(f)
	f.StringArrayVar(&w.prefix, "prefix", nil, "Key prefix segment (repeatable). Required in raw mode.")
	f.BoolVar(&w.confirm, "confirm", false, "Actually perform the deletion (without it, prints a dry-run preview).")

	return cmd
}

func (w *wipe) Run(_ *cobra.Command, _ []string) {
	storeID, scopePrefix, _ := w.scope.resolveStore(w.configFactory)

	apiPrefix := append(append([]string{}, scopePrefix...), w.prefix...)
	if err := ensureWipePrefixSafe(apiPrefix); err != nil {
		utils.NewExitError().WithMessage(err.Error()).Done()
	}

	prefixKey := strings.Join(apiPrefix, "/")

	n := countEntries(w.configFactory, storeID, apiPrefix)
	if n == 0 {
		fmt.Printf("No entries match prefix %q in store %s; nothing to delete.\n", prefixKey, storeID)

		return
	}

	fmt.Printf("This will delete %d %s under prefix %q in store %s.\n", n, pluralEntries(n), prefixKey, storeID)

	if !w.confirm {
		fmt.Println("Re-run with --confirm to proceed. No changes were made.")

		return
	}

	err := w.configFactory.
		NewRequest().
		WithMethod(http.MethodDelete).
		JSONBody(deleteBody{Options: deleteOptions{Cascade: true}}).
		WithPath(kvPath(storeID, prefixKey)).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to wipe KV entries").WithReason(err).Done()
	}

	fmt.Printf("Wiped %d %s under prefix %q.\n", n, pluralEntries(n), prefixKey)
}

func pluralEntries(n int) string {
	if n == 1 {
		return "entry"
	}

	return "entries"
}
