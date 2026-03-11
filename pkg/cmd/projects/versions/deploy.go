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

package versions

import (
	"net/http"

	"github.com/spf13/cobra"

	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/cmd/flags"
	"github.com/versori/cli/pkg/utils"
)

type Deploy struct {
	configFactory *config.ConfigFactory
	projectId     flags.ProjectId
	environment   string
	versionId     string
}

func NewDeploy(c *config.ConfigFactory) *cobra.Command {
	p := &Deploy{configFactory: c}

	cmd := &cobra.Command{
		Use:     "deploy --project <project-id> --version <version-id> --environment <environment-name>",
		Short:   "Creates a new version for a project with the files from --directory",
		Aliases: []string{"push"},
		Run:     p.Run,
	}

	flags := cmd.Flags()
	p.projectId.SetFlag(flags)
	flags.StringVar(&p.environment, "environment", "", "The name of the environment to deploy to.")
	flags.StringVar(&p.versionId, "version-id", "", "The ID of the version to deploy.")

	_ = cmd.MarkFlagRequired("version-id")
	_ = cmd.MarkFlagRequired("environment")

	return cmd
}

func (p *Deploy) Run(_ *cobra.Command, _ []string) {
	projectId := p.projectId.GetFlagOrDie(".")
	requestPath := "o/:organisation/projects/" + projectId + "/versions/" + p.versionId + "/deploy"

	err := p.configFactory.
		NewRequest().
		WithMethod(http.MethodPut).
		WithQueryParam("project_env", p.environment).
		WithPath(requestPath).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to deploy version").WithReason(err).Done()
	}
}
