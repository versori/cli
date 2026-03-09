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

type add struct {
	configFactory *config.ConfigFactory
	name          string
	organisation  string
	jwt           string
}

func NewAdd(c *config.ConfigFactory) *cobra.Command {
	a := &add{
		configFactory: c,
	}
	cmd := &cobra.Command{
		Use:   "add --name <name> --organisation <organisation-id> --jwt <jwt>",
		Short: "Add a new context to your config and selects it as the default. It requires you generate a JWT token from the Versori console. You can generate the JWT here https://ai.versori.com/account?content=keys",
		Run:   a.Run,
	}

	flags := cmd.Flags()
	flags.StringVar(&a.name, "name", "", "Name of the context")
	flags.StringVar(&a.organisation, "organisation", "", "Organisation ID to use with the context")
	flags.StringVar(&a.jwt, "jwt", "", "JWT token to use with the context. If the value is -, it will be read from stdin.")

	return cmd
}

func (a *add) Run(cmd *cobra.Command, _ []string) {
	if a.name == "" {
		nameEditor := elements.NewEditor("Enter context name:", false)
		err := nameEditor.Edit(&a.name)
		if err != nil {
			utils.NewExitError().WithMessage("failed to read context name").WithReason(err).Done()
		}
	}

	if a.organisation == "" {
		orgEditor := elements.NewEditor("Enter organisation ID:", false, elements.WithValidation(utils.IsValidULID))
		err := orgEditor.Edit(&a.organisation)
		if err != nil {
			utils.NewExitError().WithMessage("failed to read organisation ID").WithReason(err).Done()
		}
	}

	if a.jwt == "" {
		jwtEditor := elements.NewEditor("Enter JWT:", false, elements.WithValidation(utils.IsValidJWT))
		err := jwtEditor.Edit(&a.jwt)
		if err != nil {
			utils.NewExitError().WithMessage("failed to read jwt").WithReason(err).Done()
		}
	}

	if a.jwt == "-" {
		b, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			utils.NewExitError().WithMessage("failed to read jwt from stdin").WithReason(err).Done()
		}

		a.jwt = strings.TrimSpace(string(b))
	}

	if a.jwt == "" {
		utils.NewExitError().WithMessage("jwt is required").Done()
	}

	a.configFactory.AddContext(config.Context{
		Name:           a.name,
		OrganisationId: a.organisation,
		JWT:            a.jwt,
	})
}
