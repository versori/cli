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
	"github.com/spf13/cobra"

	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/cmd/projects/assets"
)

// NewAssetsCommand creates the `versori projects asset` parent command and wires its subcommands.
func NewAssetsCommand(c *config.ConfigFactory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "asset",
		Aliases: []string{"assets"},
		Short:   "Manage assets within a project",
	}

	cmd.AddCommand(assets.NewUpload(c))
	cmd.AddCommand(assets.NewList(c))
	cmd.AddCommand(assets.NewDownload(c))

	return cmd
}

