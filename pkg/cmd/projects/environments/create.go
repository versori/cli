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
	"net/http"

	"github.com/spf13/cobra"
	"github.com/versori/cli/pkg/cmd/flags"

	v1 "github.com/versori/cli/pkg/api/v1"
	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/utils"
)

// create implements `versori projects environments create`.
type create struct {
	configFactory           *config.ConfigFactory
	projectId               flags.ProjectId
	oldEnvName              string
	newEnvName              string
	executionPool           string
	cloneSystems            bool
	copyStaticUserVariables bool
}

type printableEnvironment struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Status        string `json:"status"`
	PublicURL     string `json:"publicUrl"`
	ExecutionPool string `json:"executionPool"`
}

func init() {
	utils.RegisterResource(printableEnvironment{}, []string{"ID", "Name", "Status", "PublicURL", "ExecutionPool"})
}

// NewCreate returns the cobra command for creating an environment by cloning an existing one.
func NewCreate(c *config.ConfigFactory) *cobra.Command {
	cr := &create{configFactory: c}

	cmd := &cobra.Command{
		Use:   "create --project <project-id> --old-env <source-env> --new-env <new-env>",
		Short: "Create a new environment by cloning an existing environment in the same project",
		Long: `Create a new environment by cloning an existing environment within a project.
This command creates a new environment by copying the details of an existing environment.
You can optionally clone systems and copy static user variables from the source environment.`,
		Run: cr.Run,
	}

	flags := cmd.Flags()
	cr.projectId.SetFlag(flags)
	flags.StringVar(&cr.oldEnvName, "old-env", "", "The name of the source environment to clone from")
	flags.StringVar(&cr.newEnvName, "new-env", "", "The name of the new environment to create")
	flags.StringVar(&cr.executionPool, "execution-pool", "", "Optional override execution pool for the new environment (defaults to source environment's execution pool)")
	flags.BoolVar(&cr.cloneSystems, "clone-systems", false, "If true, copies systems from the source environment to the new environment")
	flags.BoolVar(&cr.copyStaticUserVariables, "copy-static-user-variables", false, "If true, copies static user variables from the source environment to the new environment")

	_ = cmd.MarkFlagRequired("project")
	_ = cmd.MarkFlagRequired("old-env")
	_ = cmd.MarkFlagRequired("new-env")

	return cmd
}

func (c *create) Run(cmd *cobra.Command, args []string) {
	projectId := c.projectId.GetFlagOrDie(".")

	payload := v1.CloneEnvironmentRequest{
		OldEnvName:              c.oldEnvName,
		NewEnvName:              c.newEnvName,
		ExecutionPool:           c.executionPool,
		CloneSystems:            c.cloneSystems,
		CopyStaticUserVariables: c.copyStaticUserVariables,
	}

	resp := v1.Project{}
	err := c.configFactory.
		NewRequest().
		WithMethod(http.MethodPost).
		WithPath("o/:organisation/projects/" + projectId + "/environments/clone").
		Into(&resp).
		JSONBody(payload).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to create environment").WithReason(err).Done()
	}

	// Find the newly created environment in the response
	var newEnv *v1.ProjectEnvironment
	for i := range resp.Environments {
		if resp.Environments[i].Name == c.newEnvName {
			newEnv = &resp.Environments[i]

			break
		}
	}

	if newEnv == nil {
		utils.NewExitError().WithMessage("environment created but not found in response").Done()

		// this return is never hit because the exit error will exit the program on .Done()
		// this is only here to appease the linter
		return
	}

	result := printableEnvironment{
		ID:            newEnv.ID.String(),
		Name:          newEnv.Name,
		Status:        newEnv.Status,
		PublicURL:     newEnv.PublicUrl,
		ExecutionPool: newEnv.ExecutionPool,
	}

	c.configFactory.Print(result)
}
