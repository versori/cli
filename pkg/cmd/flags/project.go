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
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/pflag"
	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/utils"
)

type ProjectId string

func (p *ProjectId) SetFlag(flags *pflag.FlagSet) {
	flags.StringVar((*string)(p), "project", "", "Project ID; defaults from .versori when inside a synced project directory.")
}

// GetFlagOrDie returns the project ID from the .versori file or the project flag.
// If there is no flag set, it exits with an error.
func (p *ProjectId) GetFlagOrDie(dir string) string {
	projectId := p.GetProjectIDFromDir(dir)
	if projectId == "" {
		utils.NewExitError().WithMessage("project id not provided and not found in .versori (provide --project or run sync first)").Done()
	}

	return projectId
}

// GetProjectIDFromDir returns the project ID from the .versori file or the project flag.
// If .versori does not exist, the flag value is returned (empty string if unset).
// If .versori exists, its context must match the current context or the process exits with an error.
// If .versori exists and the flag is explicitly set to a differing value, the process exits with an error.
// If neither are set, it returns an empty string.
func (p *ProjectId) GetProjectIDFromDir(dir string) string {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		utils.NewExitError().WithMessage("failed to resolve directory").WithReason(err).Done()
	}

	versoriPath := filepath.Join(absDir, ".versori")

	v, err := ReadVersoriConfig(versoriPath)
	if err != nil {
		if os.IsNotExist(err) {
			// p is nil or empty so default to empty string
			// it will be up to the caller to decide what to do with it
			if p == nil {
				return ""
			}

			// return the flag value if file is not found
			return string(*p)
		}

		utils.NewExitError().WithMessage("failed to read .versori").WithReason(err).Done()
	}

	if config.CurrentContext == nil {
		utils.NewExitError().WithMessage("config not loaded; ensure LoadConfigAndContext() is called before GetProjectIDFromDir()").Done()
	}

	currentContext := config.CurrentContext.Name
	if v.Context != currentContext {
		utils.NewExitError().WithMessage(fmt.Sprintf("current context %q does not match .versori context %q", currentContext, v.Context)).Done()
	}

	// Only validate the flag value if it was explicitly provided.
	if p != nil && *p != "" && string(*p) != v.ProjectId {
		utils.NewExitError().WithMessage(fmt.Sprintf("--project %q does not match .versori project %q", *p, v.ProjectId)).Done()
	}

	// .versori is the source of truth once we reach here.
	return v.ProjectId
}
