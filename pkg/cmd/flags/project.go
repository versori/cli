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
	"github.com/versori/cli/pkg/versorifile"
)

type ProjectId string

func (p *ProjectId) SetFlag(flags *pflag.FlagSet) {
	flags.StringVar((*string)(p), "project", "", "Project ID; defaults from .versori when inside a synced project directory.")
}

// GetFlagOrDie returns the project ID from the .versori file or the project flag.
// If there is no flag set, it exits with an error.
//
// As a side-effect, when the resolved project ID matches the .versori file's
// project_id, the CLI context is switched to the one pinned by .versori so
// subsequent API calls hit the right org without the user having to run
// `versori context select` first. See config.MaybeApplyVersoriContextForProject
// for the full precedence rules.
func (p *ProjectId) GetFlagOrDie(dir string) string {
	projectId := p.GetProjectIDFromDir(dir)
	if projectId == "" {
		utils.NewExitError().WithMessage("project id not provided and not found in .versori (provide --project or run sync first)").Done()
	}

	config.MaybeApplyVersoriContextForProject(dir, projectId)

	return projectId
}

// GetProjectIDFromDir resolves the project ID for a command using the following precedence:
//
//  1. No .versori in dir: the --project flag value is returned (empty string if unset).
//  2. .versori present, --project unset: the file's project_id is returned.
//  3. .versori present, --project set: --project WINS. A warning is written to stderr
//     when the two differ so accidental cross-project work stays visible.
//
// Pure read — never mutates the CLI context. Use GetFlagOrDie (or call
// config.MaybeApplyVersoriContextForProject explicitly) if you also want the
// .versori-bound context to take effect.
func (p *ProjectId) GetProjectIDFromDir(dir string) string {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		utils.NewExitError().WithMessage("failed to resolve directory").WithReason(err).Done()
	}

	v, err := versorifile.FromDir(absDir)
	if err != nil {
		utils.NewExitError().WithMessage("failed to read .versori").WithReason(err).Done()
	}

	if v == nil {
		// No .versori file: defer to the flag (or empty if unset / nil receiver).
		if p == nil {
			return ""
		}

		return string(*p)
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
