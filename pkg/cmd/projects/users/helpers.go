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

package users

import (
	"net/http"

	v1 "github.com/versori/cli/pkg/api/v1"
	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/utils"
)

// resolveActivationID looks up the activation ID for the given end-user external ID
// on the given environment. Exits with an error if no matching activation is found.
func resolveActivationID(cf *config.ConfigFactory, envId, userExternalId string) string {
	var activations []v1.Activation
	err := cf.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(&activations).
		WithPath("o/:organisation/environments/" + envId + "/activations").
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to list activations").WithReason(err).Done()
	}

	for _, a := range activations {
		if a.User.ExternalID == userExternalId {
			return a.ID.String()
		}
	}

	utils.NewExitError().WithMessage("no activation found for end-user external ID: " + userExternalId).Done()

	return "" // unreachable
}
