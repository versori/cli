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

package flags

import (
	"encoding/json"
	"os"
)

// VersoriFile is the shape of the .versori file in a synced project directory.
type VersoriFile struct {
	ProjectId string `json:"project_id"`
	Context   string `json:"context"`
}

func ReadVersoriConfig(path string) (*VersoriFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var v VersoriFile
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}

	return &v, nil
}

func WriteVersoriConfig(path string, v *VersoriFile) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o600)
}
