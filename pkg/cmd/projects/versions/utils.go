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

	v1 "github.com/versori/cli/pkg/api/v1"
	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/cmd/elements"
	"github.com/versori/cli/pkg/utils"
)

func selectProject(c *config.ConfigFactory, value *string) {
	projects := v1.ProjectsList{}
	err := c.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(&projects).
		WithPath("o/:organisation/projects").
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to list projects").WithReason(err).Done()
	}

	projectSelect := elements.NewListSelect("Select a project:")

	for _, proj := range projects.Projects {
		projectSelect.AddOption(proj.Name, proj.ID.String())
	}

	err = projectSelect.Select(value)
	if err != nil {
		utils.NewExitError().WithMessage("failed to select project").WithReason(err).Done()
	}
}

func selectVersion(c *config.ConfigFactory, projectId string, value *string) {
	var versionPage v1.VersionPage
	err := c.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(&versionPage).
		WithPath("o/:organisation/projects/" + projectId + "/versions").
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to list versions").WithReason(err).Done()
	}
	versionSelect := elements.NewListSelect("Select a version to pull:")

	for _, v := range versionPage.Items {
		versionSelect.AddOption(v.Name, v.ID.String())
	}

	err = versionSelect.Select(value)
	if err != nil {
		utils.NewExitError().WithMessage("failed to select version").WithReason(err).Done()
	}
}
