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

	v1 "github.com/versori/cli/pkg/api/v1"
	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/utils"
)

// promote implements `versori projects environments promote`.
type promote struct {
	configFactory *config.ConfigFactory
	projectId     string
	sourceEnv     string
	targetEnv     string
}

type printableProject struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Environments []string `json:"environments"`
}

func init() {
	utils.RegisterResource(printableProject{}, []string{"ID", "Name", "Environments"})
}

// NewPromote returns the cobra command for promoting one environment to another.
func NewPromote(c *config.ConfigFactory) *cobra.Command {
	pr := &promote{configFactory: c}

	cmd := &cobra.Command{
		Use:   "promote --project <project-id> --source <source-env> --target <target-env>",
		Short: "Promote one environment to another by syncing deployment configuration",
		Long: `Promote one environment to another within a project.
This command copies the deployment configuration (currently the deployed docker image/version) 
from the source environment to the target environment.

For example, to promote staging to production:
  versori projects environments promote --project <id> --source staging --target production`,
		Run: pr.Run,
	}

	flags := cmd.Flags()
	flags.StringVar(&pr.projectId, "project", "", "The ID of the project")
	flags.StringVar(&pr.sourceEnv, "source", "", "The name of the source environment to promote from")
	flags.StringVar(&pr.targetEnv, "target", "", "The name of the target environment to promote to")

	_ = cmd.MarkFlagRequired("project")
	_ = cmd.MarkFlagRequired("source")
	_ = cmd.MarkFlagRequired("target")

	return cmd
}

func (p *promote) Run(cmd *cobra.Command, args []string) {
	payload := v1.SyncEnvironmentsRequest{
		SourceEnvName: p.sourceEnv,
		TargetEnvName: p.targetEnv,
	}

	resp := v1.Project{}
	err := p.configFactory.
		NewRequest().
		WithMethod(http.MethodPost).
		WithPath("o/:organisation/projects/" + p.projectId + "/environments/sync").
		Into(&resp).
		JSONBody(payload).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to promote environment").WithReason(err).Done()
	}

	// Extract environment names for display
	envNames := make([]string, 0, len(resp.Environments))
	for _, env := range resp.Environments {
		envNames = append(envNames, env.Name)
	}

	result := printableProject{
		ID:           resp.ID.String(),
		Name:         resp.Name,
		Environments: envNames,
	}

	p.configFactory.Print(result)
}
