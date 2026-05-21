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

// GetProjectIDFromDir resolves the project ID for a command using the following precedence:
//
//  1. No .versori in dir: the --project flag value is returned (empty string if unset).
//  2. .versori present, --project unset: the file's project_id is returned.
//  3. .versori present, --project set: --project WINS. A warning is written to stderr
//     when the two differ so accidental cross-project work stays visible.
//  4. .versori's context differs from the active CLI context: the active context wins.
//     A warning is written to stderr (the project_id may still be valid in the active
//     context, e.g. a project that's been re-synced under a new context).
func (p *ProjectId) GetProjectIDFromDir(dir string) string {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		utils.NewExitError().WithMessage("failed to resolve directory").WithReason(err).Done()
	}

	versoriPath := filepath.Join(absDir, ".versori")

	v, err := ReadVersoriConfig(versoriPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No .versori file: defer to the flag (or empty if unset / nil receiver).
			if p == nil {
				return ""
			}

			return string(*p)
		}

		utils.NewExitError().WithMessage("failed to read .versori").WithReason(err).Done()
	}

	if config.CurrentContext == nil {
		utils.NewExitError().WithMessage("config not loaded; ensure LoadConfigAndContext() is called before GetProjectIDFromDir()").Done()
	}

	currentContext := config.CurrentContext.Name
	if v.Context != currentContext {
		fmt.Fprintf(os.Stderr, "warning: active context %q overrides .versori context %q (in %s)\n",
			currentContext, v.Context, absDir)
	}

	// Explicit --project wins over .versori; warn on divergence so it doesn't go unnoticed.
	if p != nil && *p != "" {
		if string(*p) != v.ProjectId {
			fmt.Fprintf(os.Stderr, "warning: --project %q overrides .versori project %q (in %s)\n",
				string(*p), v.ProjectId, absDir)
		}

		return string(*p)
	}

	return v.ProjectId
}
