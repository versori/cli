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
	"testing"

	"github.com/spf13/cobra"
	v1 "github.com/versori/cli/pkg/api/v1"
)

func TestEditHasResourceChanges(t *testing.T) {
	tests := []struct {
		name string
		edit edit
		want bool
	}{
		{name: "no resource flags", edit: edit{}, want: false},
		{name: "memory request", edit: edit{resourceMemoryReq: "200Mi"}, want: true},
		{name: "cpu limit", edit: edit{resourceCpuLimit: "500m"}, want: true},
		{name: "storage limit", edit: edit{ephemeralStorageLimit: "1Gi"}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.edit.hasResourceChanges(); got != tt.want {
				t.Fatalf("hasResourceChanges() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEditHasChanges(t *testing.T) {
	tests := []struct {
		name                string
		edit                edit
		markServiceAccount  bool
		want                bool
	}{
		{name: "no changes", edit: edit{}, want: false},
		{name: "resource change", edit: edit{resourceCpuReq: "100m"}, want: true},
		{name: "replicas change", edit: edit{replicas: 2}, want: true},
		{name: "max replicas change", edit: edit{maxReplicas: 3}, want: true},
		{name: "static ip change", edit: edit{staticIP: staticIPEnabled}, want: true},
		{name: "service account changed", edit: edit{}, markServiceAccount: true, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: "test"}
			cmd.Flags().String("service-account", "", "")

			if tt.markServiceAccount {
				if err := cmd.Flags().Set("service-account", "svc-account"); err != nil {
					t.Fatalf("failed to set service-account flag: %v", err)
				}
			}

			if got := tt.edit.hasChanges(cmd); got != tt.want {
				t.Fatalf("hasChanges() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEditBuildPayloadEphemeralStorageOnly(t *testing.T) {
	t.Run("requests only", func(t *testing.T) {
		e := &edit{ephemeralStorageReq: "1Gi"}
		current := v1.EnvironmentConfig{DeploymentSpec: &v1.DeploymentSpec{}}
		got := e.buildPayload(current, false)
		if got.DeploymentSpec.Resources.Requests == nil || got.DeploymentSpec.Resources.Requests.Storage == nil ||
			*got.DeploymentSpec.Resources.Requests.Storage != "1Gi" {
			t.Fatalf("requests.storage = %#v", got.DeploymentSpec.Resources.Requests)
		}
	})
	t.Run("limits only", func(t *testing.T) {
		e := &edit{ephemeralStorageLimit: "2Gi"}
		current := v1.EnvironmentConfig{DeploymentSpec: &v1.DeploymentSpec{}}
		got := e.buildPayload(current, false)
		if got.DeploymentSpec.Resources.Limits == nil || got.DeploymentSpec.Resources.Limits.Storage == nil ||
			*got.DeploymentSpec.Resources.Limits.Storage != "2Gi" {
			t.Fatalf("limits.storage = %#v", got.DeploymentSpec.Resources.Limits)
		}
	})
}