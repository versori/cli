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
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/versori/cli/pkg/cmd/config"
)

func Execute() {
	rootCmd := GetRootCommand()

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func GetRootCommand() *cobra.Command {
	var rootCmd = &cobra.Command{
		Use:          "versori",
		Short:        "versori CLI",
		Long:         "versori CLI",
		SilenceUsage: true,
		Version:      version,
	}

	rootFlags := rootCmd.PersistentFlags()

	configFactory := config.NewConfigFactory()
	configFactory.AddFlags(rootFlags)

	rootCmd.AddCommand(versionCmd)

	// context command
	ctxCommand := newCtxCommand(configFactory)
	rootCmd.AddCommand(ctxCommand)

	// projects command
	projectsCommand := newProjectsCommand(configFactory)
	rootCmd.AddCommand(projectsCommand)

	// systems command
	systemsCommand := newSystemsCommand(configFactory)
	rootCmd.AddCommand(systemsCommand)

	// connections command
	connectionsCommand := newConnectionsCommand(configFactory)
	rootCmd.AddCommand(connectionsCommand)

	// users command
	usersCommand := newUsersCommand(configFactory)
	rootCmd.AddCommand(usersCommand)

	// execution pools
	executionPoolsCommand := newExecutionPoolsCommand(configFactory)
	rootCmd.AddCommand(executionPoolsCommand)

	// skills command
	skillsCommand := newSkillsCommand(configFactory)
	rootCmd.AddCommand(skillsCommand)

	return rootCmd
}
