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
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

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

// GetFlagOrDieDestructive is GetFlagOrDie plus a typed-confirmation prompt for
// destructive commands when --project differs from the dir's .versori project_id.
// Used by deploy/save/sync to make cross-project actions impossible to perform
// by accident: the existing stderr "warning:" line from GetProjectIDFromDir is
// easy to miss, especially in CI/agent contexts where stderr is routinely
// dropped. The prompt requires the user to type CONFIRM (or confirm) literally;
// agents/scripts can pre-acknowledge with autoConfirm=true (wired to --confirm).
//
// action is the verb shown in the prompt ("deploy", "save", "sync") and drives
// the per-command summary in crossProjectSummary.
func (p *ProjectId) GetFlagOrDieDestructive(dir, action string, autoConfirm bool) string {
	projectId := p.GetFlagOrDie(dir)
	requireCrossProjectConfirm(dir, action, projectId, autoConfirm)

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

// requireCrossProjectConfirm enforces the typed-CONFIRM gate for destructive
// commands. No-op when the dir has no .versori, when .versori already matches
// projectId (no cross-project risk), or when autoConfirm is true. Otherwise,
// prints an action-specific summary and reads a line from stdin; only "CONFIRM"
// or "confirm" (after trimming) lets the command proceed. Stdin EOF (typical of
// non-interactive callers that forgot --confirm) aborts with a hint rather than
// hanging — matches the same agent-safety stance as the rest of the CLI.
func requireCrossProjectConfirm(dir, action, projectId string, autoConfirm bool) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		utils.NewExitError().WithMessage("failed to resolve directory").WithReason(err).Done()
	}

	v, err := versorifile.FromDir(absDir)
	if err != nil {
		utils.NewExitError().WithMessage("failed to read .versori").WithReason(err).Done()
	}

	if v == nil || v.ProjectId == projectId {
		return
	}

	if autoConfirm {
		return
	}

	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "!! CROSS-PROJECT %s !!\n", strings.ToUpper(action))
	fmt.Fprintln(os.Stderr, crossProjectSummary(action, projectId, v.ProjectId, absDir))
	fmt.Fprintln(os.Stderr, "Pass --confirm to skip this prompt in scripts/agents.")
	fmt.Fprint(os.Stderr, "Type CONFIRM (or confirm) to proceed: ")

	reader := bufio.NewReader(os.Stdin)

	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		utils.NewExitError().WithMessage("failed to read confirmation").WithReason(err).Done()
	}

	if errors.Is(err, io.EOF) && strings.TrimSpace(line) == "" {
		utils.NewExitError().WithMessage("aborted: stdin closed before confirmation. Pass --confirm to skip the prompt in scripts/agents.").Done()
	}

	if line := strings.TrimSpace(line); line != "CONFIRM" && line != "confirm" {
		utils.NewExitError().WithMessage("aborted: confirmation not entered").Done()
	}
}

// crossProjectSummary renders a human-readable description of what the command
// is about to do across two projects. Direction matters: sync writes the
// remote's files OVER the local dir (and re-pins .versori), while deploy/save
// upload the local dir's files INTO a different remote project.
func crossProjectSummary(action, flagProjectId, versoriProjectId, absDir string) string {
	switch action {
	case "sync":
		return fmt.Sprintf(
			"Project %s will OVERWRITE all files in the existing project (%s) in %s, and rewrite .versori to point at %s.",
			flagProjectId, versoriProjectId, absDir, flagProjectId)
	case "deploy":
		return fmt.Sprintf(
			"Local files in %s (currently pinned to project %s) will be DEPLOYED as a new version of project %s.",
			absDir, versoriProjectId, flagProjectId)
	case "save":
		return fmt.Sprintf(
			"Local files in %s (currently pinned to project %s) will be SAVED to project %s.",
			absDir, versoriProjectId, flagProjectId)
	default:
		return fmt.Sprintf(
			"%s will run against project %s using files from %s (currently pinned to project %s).",
			action, flagProjectId, absDir, versoriProjectId)
	}
}
