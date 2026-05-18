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

package variables

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/cmd/flags"
	"github.com/versori/cli/pkg/utils"
)

type get struct {
	configFactory *config.ConfigFactory
	projectId     flags.ProjectId
}

func NewGet(c *config.ConfigFactory) *cobra.Command {
	g := &get{configFactory: c}

	cmd := &cobra.Command{
		Use:   "get --project <project-id>",
		Short: "Get the project's DynamicVariablesSchema (the JSON schema defining valid activation-variable keys)",
		Long: `Fetch the JSON schema that defines which dynamic-variable keys end-user activations
on this project may set. Activations whose variables don't match this schema are rejected at
creation time. Use this to discover which keys exist before running 'versori projects users
activate' or 'versori projects users set-variable[s]'.`,
		Run: g.Run,
	}

	g.projectId.SetFlag(cmd.Flags())

	return cmd
}

func (g *get) Run(_ *cobra.Command, _ []string) {
	projectId := g.projectId.GetFlagOrDie(".")

	raw := json.RawMessage{}
	err := g.configFactory.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(&raw).
		WithPath("o/:organisation/projects/" + projectId + "/variables").
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to get DynamicVariablesSchema").WithReason(err).Done()
	}

	pretty, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		// fall back to the raw bytes if pretty-printing fails
		fmt.Println(string(raw))

		return
	}

	fmt.Println(string(pretty))
}
