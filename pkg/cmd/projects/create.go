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

package projects

import (
	"net/http"

	"github.com/spf13/cobra"

	v1 "github.com/versori/cli/pkg/api/v1"
	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/cmd/elements"
	"github.com/versori/cli/pkg/utils"
)

// create implements `versori projects create`.
type create struct {
	configFactory *config.ConfigFactory
	name          string
}

// NewCreate returns the cobra command for creating a project.
func NewCreate(c *config.ConfigFactory) *cobra.Command {
	cr := &create{configFactory: c}

	cmd := &cobra.Command{
		Use:   "create --name <name>",
		Short: "Create a new project in the current organisation",
		Run:   cr.Run,
	}

	flags := cmd.Flags()
	flags.StringVarP(&cr.name, "name", "n", "", "Name of the project")

	return cmd
}

func (c *create) Run(cmd *cobra.Command, args []string) {
	if c.name == "" {
		editor := elements.NewEditor("Enter project name:", false)
		err := editor.Edit(&c.name)
		if err != nil {
			utils.NewExitError().WithMessage("failed to read project name").WithReason(err).Done()
		}
	}

	payload := v1.ProjectCreate{
		Name: c.name,
	}

	resp := v1.Project{}
	err := c.configFactory.
		NewRequest().
		WithMethod(http.MethodPost).
		WithPath("o/:organisation/projects").
		Into(&resp).
		JSONBody(payload).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to create project").WithReason(err).Done()
	}

	summary := ProjectSummary{
		Id:           resp.ID.String(),
		Name:         resp.Name,
		Deployed:     deployed(resp.Environments),
		Environments: envNames(resp.Environments),
	}

	c.configFactory.Print(summary)
}
