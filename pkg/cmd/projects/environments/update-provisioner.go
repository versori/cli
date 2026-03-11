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

package environments

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/versori/cli/pkg/cmd/flags"

	v1 "github.com/versori/cli/pkg/api/v1"
	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/utils"
)

// updateExecutionPool implements `versori projects environments update-execution-pool`.
type updateExecutionPool struct {
	configFactory    *config.ConfigFactory
	environment      string
	projectId        flags.ProjectId
	executionPool    string
	skipConfirmation bool
}

// NewUpdateExecutionPool returns the cobra command for updating an environment's execution pool.
func NewUpdateExecutionPool(c *config.ConfigFactory) *cobra.Command {
	up := &updateExecutionPool{configFactory: c}

	cmd := &cobra.Command{
		Use:   "update-execution-pool --environment <environment-id> --execution-pool <execution-pool-name>",
		Short: "Update the execution pool of an environment",
		Long: `Update the execution pool used to deploy an environment.

WARNING: Changing the execution pool will result in a new public URL being created for the environment.
If the environment is running, it will be suspended first.

Example:
  versori projects environments update-execution-pool --environment 01ABC123 --execution-pool gcp`,
		Run: up.Run,
	}

	flags := cmd.Flags()

	up.projectId.SetFlag(flags)
	flags.StringVar(&up.environment, "environment", "", "The name of the environment to update")
	flags.StringVar(&up.executionPool, "execution-pool", "", "The name of the new execution pool")
	flags.BoolVarP(&up.skipConfirmation, "yes", "y", false, "Skip confirmation prompt")

	_ = cmd.MarkFlagRequired("environment")
	_ = cmd.MarkFlagRequired("execution-pool")

	return cmd
}

func (u *updateExecutionPool) Run(cmd *cobra.Command, args []string) {
	projectId := u.projectId.GetFlagOrDie(".")

	// Show warning and prompt for confirmation unless --yes flag is provided
	if !u.skipConfirmation {
		fmt.Println("\n   WARNING: Changing the execution pool will change the environment's public URL.")
		fmt.Println("   If the environment is running, it will be suspended first.")
		fmt.Print("\nDo you want to continue? (yes/no): ")

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			utils.NewExitError().WithMessage("failed to read user input").WithReason(err).Done()
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response != "yes" && response != "y" {
			fmt.Println("Operation cancelled.")
			os.Exit(0)
		}
	}

	payload := v1.ChangeEnvironmentExecutionPoolRequest{
		ExecutionPool: u.executionPool,
	}

	resp := v1.ProjectEnvironment{}
	err := u.configFactory.
		NewRequest().
		WithMethod(http.MethodPut).
		WithPath("o/:organisation/projects/"+projectId+"/environments/execution-pools").
		WithQueryParam("project_env", u.environment).
		Into(&resp).
		JSONBody(payload).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to update environment execution pool").WithReason(err).Done()
	}

	result := printableEnvironment{
		ID:            resp.ID.String(),
		Name:          resp.Name,
		Status:        resp.Status,
		PublicURL:     resp.PublicUrl,
		ExecutionPool: resp.ExecutionPool,
	}

	u.configFactory.Print(result)
}
