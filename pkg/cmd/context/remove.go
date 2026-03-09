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
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/cmd/elements"
	"github.com/versori/cli/pkg/utils"
)

type remove struct {
	configFactory *config.ConfigFactory
}

func NewRemove(c *config.ConfigFactory) *cobra.Command {
	rm := &remove{
		configFactory: c,
	}

	cmd := &cobra.Command{
		Use:   "rm",
		Short: "Remove a configured context.",
		Run:   rm.Run,
	}

	return cmd
}

func (rm *remove) Run(cmd *cobra.Command, args []string) {
	rm.configFactory.LoadConfigAndContext()
	var name string

	if len(args) == 0 {
		ctxSelect := elements.NewListSelect("Select context to remove:")
		for _, ctx := range rm.configFactory.Config.Contexts {
			ctxSelect.AddOption(ctx.Name, ctx.Name)
		}

		err := ctxSelect.Select(&name)
		if err != nil {
			utils.NewExitError().WithMessage("failed to read context name").WithReason(err).Done()
		}
	} else {
		name = args[0]
	}

	if name == "-" {
		b, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			utils.NewExitError().WithMessage("failed to read context name from stdin").WithReason(err).Done()
		}

		name = strings.TrimSpace(string(b))
	}

	rm.configFactory.RemoveContext(name)
}
