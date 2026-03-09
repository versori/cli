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

package context

import (
	"github.com/spf13/cobra"

	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/utils"
)

type list struct {
	configFactory *config.ConfigFactory
}

func init() {
	utils.RegisterResource(config.Context{}, []string{"Name", "OrganisationId"})
}

func NewList(c *config.ConfigFactory) *cobra.Command {
	l := &list{
		configFactory: c,
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List currently configured contexts. * denotes the active context.",
		Run:   l.Run,
	}

	return cmd
}

func (l *list) Run(cmd *cobra.Command, args []string) {
	l.configFactory.LoadConfigAndContext()

	contextSlice := make([]config.Context, 0, len(l.configFactory.Config.Contexts))
	for _, c := range l.configFactory.Config.Contexts {
		if c.Name == l.configFactory.Config.ActiveContext {
			c.Name = "*" + c.Name
		}

		contextSlice = append(contextSlice, c)
	}

	l.configFactory.Print(contextSlice)
}
