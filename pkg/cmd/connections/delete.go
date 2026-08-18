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

package connections

import (
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"

	v1 "github.com/versori/cli/pkg/api/v1"
	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/cmd/elements"
	"github.com/versori/cli/pkg/cmd/flags"
	"github.com/versori/cli/pkg/ulid"
	"github.com/versori/cli/pkg/utils"
)

type unlink struct {
	configFactory *config.ConfigFactory
	connectionID  string
	templateID    string
	yes           bool
}

func NewUnlink(c *config.ConfigFactory) *cobra.Command {
	u := &unlink{configFactory: c}

	cmd := &cobra.Command{
		Use:   "unlink --id <connection-id> --template-id <template-id> [--yes]",
		Short: "Unlink a connection from an environment template",
		Long: `Unlink a connection from an environment (DELETE /o/{organisation}/connections/{id}/link).

The connection itself is kept. Pass --yes or --confirm in non-interactive shells; the VS Code
extension always passes --yes after its own confirmation modal.`,
		Run: u.Run,
	}

	f := cmd.Flags()
	f.StringVar(&u.connectionID, "id", "", "ULID of the connection to unlink")
	f.StringVar(&u.templateID, "template-id", "", "Connection template ID (environment system) to unlink from")
	flags.AddSkipPromptFlags(f, &u.yes)

	_ = cmd.MarkFlagRequired("id")
	_ = cmd.MarkFlagRequired("template-id")

	return cmd
}

func (u *unlink) Run(_ *cobra.Command, _ []string) {
	if err := confirmMutation(u.yes, fmt.Sprintf("Unlink connection %s from template %s?", u.connectionID, u.templateID)); err != nil {
		utils.NewExitError().WithMessage(err.Error()).Done()
	}

	templateUlid, err := ulid.Parse(u.templateID)
	if err != nil {
		utils.NewExitError().WithMessage("template-id must be a valid ULID").WithReason(err).Done()
	}

	req := v1.UnlinkConnectionFromEnvironmentJSONRequestBody{
		EnvironmentSystemID: templateUlid,
	}

	err = u.configFactory.
		NewRequest().
		WithMethod(http.MethodDelete).
		WithPath("o/:organisation/connections/" + u.connectionID + "/link").
		JSONBody(req).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to unlink connection").WithReason(err).Done()
	}

	fmt.Println("Connection unlinked from environment.")
}

type deleteConnection struct {
	configFactory *config.ConfigFactory
	connectionID  string
	yes           bool
}

func NewDelete(c *config.ConfigFactory) *cobra.Command {
	d := &deleteConnection{configFactory: c}

	cmd := &cobra.Command{
		Use:     "delete --id <connection-id> [--yes]",
		Aliases: []string{"rm", "remove"},
		Short:   "Delete a connection",
		Long: `Delete a connection (DELETE /o/{organisation}/connections/{id}).

This removes the connection itself. Unlink it from an environment first if you
only want to clear the active connection. Pass --yes or --confirm in non-interactive shells;
the VS Code extension always passes --yes after its own confirmation modal.`,
		Run: d.Run,
	}

	f := cmd.Flags()
	f.StringVar(&d.connectionID, "id", "", "ULID of the connection to delete")
	flags.AddSkipPromptFlags(f, &d.yes)

	_ = cmd.MarkFlagRequired("id")

	return cmd
}

func (d *deleteConnection) Run(_ *cobra.Command, _ []string) {
	if err := confirmMutation(d.yes, fmt.Sprintf("Delete connection %s?", d.connectionID)); err != nil {
		utils.NewExitError().WithMessage(err.Error()).Done()
	}

	err := d.configFactory.
		NewRequest().
		WithMethod(http.MethodDelete).
		WithPath("o/:organisation/connections/" + d.connectionID).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to delete connection").WithReason(err).Done()
	}

	fmt.Printf("Deleted connection %s.\n", d.connectionID)
}

func confirmMutation(yes bool, prompt string) error {
	if yes {
		return nil
	}
	if !isInteractive() {
		return fmt.Errorf("refusing to mutate a connection without confirmation in a non-interactive shell; pass --yes or --confirm to proceed")
	}
	confirmed := false
	if err := elements.NewConfirm(prompt).Confirm(&confirmed); err != nil {
		return fmt.Errorf("failed to read confirmation: %w", err)
	}
	if !confirmed {
		fmt.Println("Aborted; no changes were made.")
		os.Exit(0)
	}
	return nil
}

func isInteractive() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
