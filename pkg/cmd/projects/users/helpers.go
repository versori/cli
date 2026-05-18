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
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

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

// variableSchemaProbe is a minimal view of the project's DynamicVariablesSchema used for
// pre-flight validation of supplied --variable / --variables-file values. Only required[] and
// the top-level property keys matter for missing/unknown detection; deeper validation (types,
// enums, nested-object shapes, etc.) is left to the platform.
type variableSchemaProbe struct {
	Properties map[string]json.RawMessage `json:"properties"`
	Required   []string                   `json:"required"`
}

// validateActivationVariables fetches the project's DynamicVariablesSchema and returns a
// descriptive error if the supplied variables don't satisfy it — i.e. required keys are
// missing, or supplied keys aren't declared in the schema. The platform would reject the
// activation for the same reasons, but it reports one issue at a time and without pointing
// at the discovery commands; this pre-flight gives a single consolidated message up front.
//
// Network / parse failures fall through to nil so the API call still happens — the platform
// has the final say if the local check is inconclusive.
func validateActivationVariables(cf *config.ConfigFactory, projectId string, supplied v1.DynamicVariables) error {
	raw := json.RawMessage{}
	err := cf.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(&raw).
		WithPath("o/:organisation/projects/" + projectId + "/variables").
		Do()
	if err != nil {
		return nil
	}

	if len(raw) == 0 || string(raw) == "null" {
		if len(supplied) > 0 {
			return fmt.Errorf(
				"project has no DynamicVariablesSchema declared, but variables were supplied (%s).\n"+
					"  Declare them first:  versori projects variables add --project %s",
				strings.Join(sortedKeys(supplied), ", "), projectId)
		}
		return nil
	}

	var schema variableSchemaProbe
	if err := json.Unmarshal(raw, &schema); err != nil {
		return nil
	}

	var missing []string
	for _, key := range schema.Required {
		if _, ok := supplied[key]; !ok {
			missing = append(missing, key)
		}
	}

	var unknown []string
	for key := range supplied {
		if _, declared := schema.Properties[key]; !declared {
			unknown = append(unknown, key)
		}
	}

	if len(missing) == 0 && len(unknown) == 0 {
		return nil
	}

	var parts []string
	if len(missing) > 0 {
		sort.Strings(missing)
		parts = append(parts, fmt.Sprintf("missing required variable(s): %s", strings.Join(missing, ", ")))
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		parts = append(parts, fmt.Sprintf("unknown variable(s) not in schema: %s", strings.Join(unknown, ", ")))
	}

	return fmt.Errorf(
		"%s\n"+
			"  Inspect the schema:  versori projects variables list --project %s\n"+
			"  Pass values with:    --variable <key>=<value> [--variable ...] [--variables-file <path>]",
		strings.Join(parts, "; "), projectId)
}

func sortedKeys(m v1.DynamicVariables) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
