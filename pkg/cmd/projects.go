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

package cmd

import (
	"github.com/spf13/cobra"

	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/cmd/projects"
)

func newProjectsCommand(c *config.ConfigFactory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "projects",
		Aliases: []string{"project"},
		Short:   "Manage projects, deployments, versions, assets, and environments",
		Long:    "The `projects` command (alias: `project`) provides subcommands for the full project lifecycle — creation, file management, deployment, logging, and more.",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			c.LoadConfigAndContext()
		},
	}

	list := projects.NewList(c)
	sync := projects.NewSync(c)
	deploy := projects.NewDeploy(c)
	save := projects.NewSave(c)
	details := projects.NewDetails(c)
	logs := projects.NewLogs(c)
	create := projects.NewCreate(c)
	edit := projects.NewEdit(c)
	proxy := projects.NewProxy(c)
	starCmd := projects.NewStar(c)
	unstarCmd := projects.NewUnstar(c)
	filesCmd := projects.NewFiles(c)

	systems := projects.NewSystemsCommand(c)
	users := projects.NewUsersCommand(c)
	versions := projects.NewVersionsCommand(c)
	environments := projects.NewEnvironmentsCommand(c)
	assetsCmd := projects.NewAssetsCommand(c)

	cmd.AddCommand(list)
	cmd.AddCommand(sync)
	cmd.AddCommand(deploy)
	cmd.AddCommand(save)
	cmd.AddCommand(details)
	cmd.AddCommand(logs)
	cmd.AddCommand(create)
	cmd.AddCommand(edit)
	cmd.AddCommand(proxy)
	cmd.AddCommand(starCmd)
	cmd.AddCommand(unstarCmd)
	cmd.AddCommand(filesCmd)
	cmd.AddCommand(systems)
	cmd.AddCommand(users)
	cmd.AddCommand(versions)
	cmd.AddCommand(environments)
	cmd.AddCommand(assetsCmd)

	return cmd
}
