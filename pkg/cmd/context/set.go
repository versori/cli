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
	"github.com/versori/cli/pkg/cmd/elements"
	"github.com/versori/cli/pkg/utils"
)

type set struct {
	configFactory *config.ConfigFactory
}

func NewSet(c *config.ConfigFactory) *cobra.Command {
	s := set{
		configFactory: c,
	}

	cmd := &cobra.Command{
		Use:   "select <context-name>",
		Short: "Changes the active context.",
		Run:   s.Run,
	}

	return cmd
}

func (a *set) Run(cmd *cobra.Command, args []string) {
	a.configFactory.LoadConfigAndContext()
	var ctxName string

	if len(args) == 0 {
		listSelector := elements.NewListSelect("Select context to use:")
		for name := range a.configFactory.Config.Contexts {
			listSelector.AddOption(name, "")
		}
		if err := listSelector.Select(&ctxName); err != nil {
			utils.NewExitError().WithMessage("Failed to select context").WithReason(err).Done()
		}
	} else {
		ctxName = args[0]
	}

	a.configFactory.SetActiveContext(ctxName)
}
