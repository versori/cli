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

type storesList struct {
	configFactory *config.ConfigFactory
}

// NewStores builds the `kv stores` subgroup (currently just `list`).
func NewStores(c *config.ConfigFactory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stores",
		Short: "Inspect KV stores",
	}

	cmd.AddCommand(newStoresList(c))

	return cmd
}

func newStoresList(c *config.ConfigFactory) *cobra.Command {
	l := &storesList{configFactory: c}

	return &cobra.Command{
		Use:   "list",
		Short: "List KV stores in the current organisation",
		Long: `List the KV stores in the current organisation context. Stores are created lazily by the
runtime on first write and named ORG_<org>, PROJECT_<project>, or EXECUTION_<project>. Use the
returned store ID with the other kv commands' --store flag, or address a store by --scope.`,
		Run: l.Run,
	}
}

func (l *storesList) Run(_ *cobra.Command, _ []string) {
	resp := storesResponse{}
	err := l.configFactory.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(&resp).
		WithPath("o/:organisation/store").
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to list KV stores").WithReason(err).Done()
	}

	items := make([]printableStore, 0, len(resp.Stores))
	for _, st := range resp.Stores {
		items = append(items, printableStore{Name: st.Name, ID: st.ID})
	}

	l.configFactory.Print(items)
}
