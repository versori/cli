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

package variables

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"

	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/cmd/flags"
	"github.com/versori/cli/pkg/utils"
)

type set struct {
	configFactory *config.ConfigFactory
	projectId     flags.ProjectId
	file          string
}

func NewSet(c *config.ConfigFactory) *cobra.Command {
	s := &set{configFactory: c}

	cmd := &cobra.Command{
		Use:   "set --project <project-id> --file <schema.json>",
		Short: "Replace the project's DynamicVariablesSchema in full (low-level escape hatch)",
		Long: `Replace the entire JSON schema defining the valid activation-variable keys for this project.
This is a full PUT — any keys not present in the new schema will be removed.

Most users should prefer the high-level commands ('add', 'update', 'remove', 'list'), which work
in terms of variable Name / Type / Description / Required without requiring you to write raw JSON
Schema. Use 'set' only when you need advanced JSON Schema features (enum, default, nested object
shapes, patternProperties, etc.) that the high-level commands don't expose, or to bulk-import a
schema from a file in CI.

Example schema declaring two string variables:

  {
    "type": "object",
    "properties": {
      "tenant_org_id": { "type": "string" },
      "channel_id":    { "type": "string" }
    }
  }

After updating the schema, end-user activations can set those keys via 'versori projects users
activate --variable key=value' or 'versori projects users set-variable[s]'.`,
		Run: s.Run,
	}

	f := cmd.Flags()
	s.projectId.SetFlag(f)
	f.StringVarP(&s.file, "file", "f", "", "Path to a JSON file containing the full DynamicVariablesSchema")

	_ = cmd.MarkFlagRequired("file")

	return cmd
}

func (s *set) Run(_ *cobra.Command, _ []string) {
	projectId := s.projectId.GetFlagOrDie(".")

	data, err := os.ReadFile(s.file)
	if err != nil {
		utils.NewExitError().WithMessage("failed to read schema file").WithReason(err).Done()
	}

	// Validate it parses as JSON before sending so we get a clean error locally rather than from the API.
	var probe any
	if err := json.Unmarshal(data, &probe); err != nil {
		utils.NewExitError().WithMessage("schema file is not valid JSON").WithReason(err).Done()
	}

	raw := json.RawMessage(data)

	resp := json.RawMessage{}
	err = s.configFactory.
		NewRequest().
		WithMethod(http.MethodPut).
		Into(&resp).
		WithPath("o/:organisation/projects/" + projectId + "/variables").
		JSONBody(raw).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to set DynamicVariablesSchema").WithReason(err).Done()
	}

	fmt.Println("DynamicVariablesSchema updated.")
}
