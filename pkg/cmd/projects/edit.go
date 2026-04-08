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
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/versori/cli/pkg/cmd/flags"

	v1 "github.com/versori/cli/pkg/api/v1"
	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/utils"
)

const (
	staticIPEnabled  = "enabled"
	staticIPDisabled = "disabled"
)

// edit implements `versori projects edit`.
type edit struct {
	configFactory         *config.ConfigFactory
	projectId             flags.ProjectId
	environmentName       string
	resourceMemoryReq     string
	resourceMemoryLimit   string
	resourceCpuReq        string
	resourceCpuLimit      string
	ephemeralStorageReq   string
	ephemeralStorageLimit string
	replicas              int
	maxReplicas           int
	serviceAccount        string
	staticIP              string
}

// NewEdit returns the cobra command for editing a project environment configuration.
func NewEdit(c *config.ConfigFactory) *cobra.Command {
	e := &edit{configFactory: c}

	cmd := &cobra.Command{
		Use:   "edit --project <project-id> --environment <environment-name>",
		Short: "Edit project environment configuration (resource limits and requests)",
		Run:   e.Run,
	}

	flags := cmd.Flags()

	e.projectId.SetFlag(flags)
	flags.StringVar(&e.environmentName, "environment", "", "The environment name")
	flags.StringVar(&e.resourceMemoryReq, "resource.memory.requests", "", "Memory requests (e.g., 200Mi)")
	flags.StringVar(&e.resourceMemoryLimit, "resource.memory.limits", "", "Memory limits (e.g., 500Mi)")
	flags.StringVar(&e.resourceCpuReq, "resource.cpu.requests", "", "CPU requests (e.g., 100m)")
	flags.StringVar(&e.resourceCpuLimit, "resource.cpu.limits", "", "CPU limits (e.g., 500m)")
	flags.StringVar(&e.ephemeralStorageReq, "resource.storage.requests", "", "Ephemeral storage requests (e.g., 1Gi)")
	flags.StringVar(&e.ephemeralStorageLimit, "resource.storage.limits", "", "Ephemeral storage limits (e.g., 1Gi)")
	flags.IntVar(&e.replicas, "replicas", 0, "Number of replicas")
	flags.IntVar(&e.maxReplicas, "max-replicas", 0, "Maximum number of replicas. Setting this option enables autoscaling on the project")
	flags.StringVar(&e.serviceAccount, "service-account", "", "Service account to use for the environment. Pass an empty string to remove the service account")
	flags.StringVar(&e.staticIP, "static-ip", "", "Enable or disable static IP (enabled/disabled)")

	_ = cmd.MarkFlagRequired("environment")

	return cmd
}

func (e *edit) Run(cmd *cobra.Command, args []string) {
	projectId := e.projectId.GetFlagOrDie(".")

	// Validate static-ip flag if provided
	if e.staticIP != "" && e.staticIP != staticIPEnabled && e.staticIP != staticIPDisabled {
		utils.NewExitError().WithMessage("--static-ip must be either 'enabled' or 'disabled'").Done()
	}

	serviceAccountChanged := cmd.Flags().Changed("service-account")
	if !e.hasChanges(cmd) {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "nothing to do")

		return
	}

	// Fetch project to resolve environment ID from name
	project := v1.Project{}
	err := e.configFactory.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(&project).
		WithPath("o/:organisation/projects/" + projectId).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to get project").WithReason(err).Done()
	}

	// Resolve environment ID by name and get current config
	envId := ""
	var currentConfig v1.EnvironmentConfig
	for _, env := range project.Environments {
		if env.Name == e.environmentName {
			envId = env.ID.String()
			currentConfig = env.Config

			break
		}
	}
	if envId == "" {
		utils.NewExitError().WithMessage("environment [" + e.environmentName + "] not found in project").Done()
	}

	// Build the request payload by merging with current config
	payload := e.buildPayload(currentConfig, serviceAccountChanged)

	// Make the PATCH request
	err = e.configFactory.
		NewRequest().
		WithMethod(http.MethodPut).
		WithPath("o/:organisation/environments/" + envId + "/config").
		JSONBody(payload).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to update environment configuration").WithReason(err).Done()
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "environment configuration updated successfully")
}

func (e *edit) hasChanges(cmd *cobra.Command) bool {
	hasResourceChanges := e.hasResourceChanges()
	hasReplicaChanges := e.replicas > 0 || e.maxReplicas > 0
	hasStaticIPChange := e.staticIP != ""
	hasServiceAccountChange := cmd.Flags().Changed("service-account")

	return hasResourceChanges || hasReplicaChanges || hasStaticIPChange || hasServiceAccountChange
}

func (e *edit) hasResourceChanges() bool {
	return e.resourceMemoryReq != "" || e.resourceMemoryLimit != "" ||
		e.resourceCpuReq != "" || e.resourceCpuLimit != "" ||
		e.ephemeralStorageReq != "" || e.ephemeralStorageLimit != ""
}

func (e *edit) buildPayload(currentConfig v1.EnvironmentConfig, serviceAccountChanged bool) v1.EnvironmentConfig {
	if currentConfig.DeploymentSpec == nil {
		currentConfig.DeploymentSpec = &v1.DeploymentSpec{}
	}

	// Start with the current config
	payload := currentConfig

	// Update replicas if flag was set (positive value)
	if e.replicas > 0 {
		payload.DeploymentSpec.Replicas = &e.replicas
	}

	if e.maxReplicas > 0 {
		payload.DeploymentSpec.Autoscaling = &v1.DeploymentAutoscaling{MaxReplicas: &e.maxReplicas}
	} else {
		payload.DeploymentSpec.Autoscaling = nil
	}

	// Update service account if flag was explicitly provided
	if serviceAccountChanged {
		if e.serviceAccount == "" {
			payload.DeploymentSpec.ServiceAccountName = nil
		} else {
			payload.DeploymentSpec.ServiceAccountName = &e.serviceAccount
		}
	}

	// Handle static IP configuration
	switch e.staticIP {
	case staticIPEnabled:
		e.enableStaticIP(&payload)
	case staticIPDisabled:
		e.disableStaticIP(&payload)
	}

	// Ensure deployment spec exists
	if payload.DeploymentSpec.Resources == nil {
		payload.DeploymentSpec.Resources = &v1.Resources{}
	}

	// Ensure limits exists if we're setting any limit
	if e.resourceCpuLimit != "" || e.resourceMemoryLimit != "" || e.ephemeralStorageLimit != "" {
		if payload.DeploymentSpec.Resources.Limits == nil {
			payload.DeploymentSpec.Resources.Limits = &v1.ResourceRequirements{}
		}
		// Update limits
		if e.resourceCpuLimit != "" {
			payload.DeploymentSpec.Resources.Limits.Cpu = &e.resourceCpuLimit
		}
		if e.resourceMemoryLimit != "" {
			payload.DeploymentSpec.Resources.Limits.Memory = &e.resourceMemoryLimit
		}
		if e.ephemeralStorageLimit != "" {
			payload.DeploymentSpec.Resources.Limits.Storage = &e.ephemeralStorageLimit
		}
	}

	// Ensure requests exists if we're setting any request
	if e.resourceCpuReq != "" || e.resourceMemoryReq != "" || e.ephemeralStorageReq != "" {
		if payload.DeploymentSpec.Resources.Requests == nil {
			payload.DeploymentSpec.Resources.Requests = &v1.ResourceRequirements{}
		}
		// Update requests
		if e.resourceCpuReq != "" {
			payload.DeploymentSpec.Resources.Requests.Cpu = &e.resourceCpuReq
		}
		if e.resourceMemoryReq != "" {
			payload.DeploymentSpec.Resources.Requests.Memory = &e.resourceMemoryReq
		}
		if e.ephemeralStorageReq != "" {
			payload.DeploymentSpec.Resources.Requests.Storage = &e.ephemeralStorageReq
		}
	}

	return payload
}

func (e *edit) enableStaticIP(payload *v1.EnvironmentConfig) {
	// Set static IP to true
	payload.DeploymentSpec.StaticIP = utils.Ptr(true)
}

func (e *edit) disableStaticIP(payload *v1.EnvironmentConfig) {
	// Remove affinity and tolerations
	payload.DeploymentSpec.StaticIP = utils.Ptr(false)
}
