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

type del struct {
	configFactory *config.ConfigFactory
	scope         kvScope
	key           string
	yes           bool
}

// NewDelete builds `kv delete` — remove a single key. MUTATION: changes live workflow state. Only
// run when the user explicitly asks to delete a specific key. To remove a whole subtree use
// `kv wipe`.
func NewDelete(c *config.ConfigFactory) *cobra.Command {
	d := &del{configFactory: c}

	cmd := &cobra.Command{
		Use:     "delete (--store <id> | --scope <scope> ...) --key <a/b/c>",
		Aliases: []string{"rm"},
		Short:   "Delete a single KV entry (mutation)",
		Long: `Delete one entry by key.

WARNING: this mutates live workflow state. Only run it when explicitly asked to delete a specific
key — never as a side-effect of debugging. This removes a single key only; to delete everything
under a prefix use kv wipe.

Confirms before deleting unless --yes is passed; in non-interactive shells --yes is required.`,
		Run: d.Run,
	}

	f := cmd.Flags()
	d.scope.addFlags(f)
	f.StringVar(&d.key, "key", "", "Key to delete (slash-delimited).")
	f.BoolVarP(&d.yes, "yes", "y", false, "Skip the confirmation prompt.")

	_ = cmd.MarkFlagRequired("key")

	return cmd
}

func (d *del) Run(_ *cobra.Command, _ []string) {
	storeID, scopePrefix, _ := d.scope.resolveStore(d.configFactory)

	keyParts := splitKey(d.key)
	if len(keyParts) == 0 {
		utils.NewExitError().WithMessage("--key must not be empty").Done()
	}

	fullParts := append(append([]string{}, scopePrefix...), keyParts...)
	fullKey := strings.Join(fullParts, "/")

	confirmOrAbort(d.yes, fmt.Sprintf("Delete key %q from store %s?", fullKey, storeID))

	err := d.configFactory.
		NewRequest().
		WithMethod(http.MethodDelete).
		JSONBody(deleteBody{Options: deleteOptions{}}).
		WithPath(kvPath(storeID, fullKey)).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to delete KV entry").WithReason(err).Done()
	}

	fmt.Printf("Deleted %q.\n", fullKey)
}
