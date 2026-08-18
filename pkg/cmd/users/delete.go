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
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"

	v1 "github.com/versori/cli/pkg/api/v1"
	"github.com/versori/cli/pkg/cmd/config"
	"github.com/versori/cli/pkg/cmd/elements"
	"github.com/versori/cli/pkg/ulid"
	"github.com/versori/cli/pkg/utils"
)

type deleteUser struct {
	configFactory *config.ConfigFactory
	id            string
	externalId    string
	yes           bool
}

func NewDelete(c *config.ConfigFactory) *cobra.Command {
	d := &deleteUser{configFactory: c}

	cmd := &cobra.Command{
		Use:     "delete (--id <ulid> | --external-id <id>) [--yes]",
		Aliases: []string{"rm", "remove"},
		Short:   "Delete an end-user from the current organisation",
		Long: `Delete an end-user from the current organisation (DELETE /o/{organisation}/users/{user_id}).

This removes the end-user record itself, not just an activation on one environment. Activations
and embedded connections owned by that user are also removed by the platform.

Pass --id (the user ULID) or --external-id (resolved client-side). Confirms before deleting
unless --yes is passed; in a non-interactive shell --yes is required.`,
		Run: d.Run,
	}

	flags := cmd.Flags()
	flags.StringVar(&d.id, "id", "", "ULID of the end-user to delete")
	flags.StringVarP(&d.externalId, "external-id", "e", "", "External ID of the end-user to delete (resolved to a ULID)")
	flags.BoolVarP(&d.yes, "yes", "y", false, "Skip the confirmation prompt")

	return cmd
}

func (d *deleteUser) Run(_ *cobra.Command, _ []string) {
	target := d.id
	if target == "" {
		target = d.externalId
	}
	if target == "" {
		utils.NewExitError().WithMessage("pass --id or --external-id").Done()
	}

	users := d.fetchUsers()
	userID, err := resolveDeleteUserID(target, users)
	if err != nil {
		utils.NewExitError().WithMessage(err.Error()).Done()
	}

	label := target
	for _, u := range users {
		if u.ID.String() == userID {
			label = fmt.Sprintf("%s (%s)", u.DisplayName, u.ExternalID)
			break
		}
	}

	if !d.yes {
		if !isInteractive() {
			utils.NewExitError().
				WithMessage("refusing to delete a user without confirmation in a non-interactive shell; pass --yes to proceed").
				Done()
		}

		confirmed := false
		err := elements.
			NewConfirm(fmt.Sprintf("Delete end-user %s from this organisation?", label)).
			Confirm(&confirmed)
		if err != nil {
			utils.NewExitError().WithMessage("failed to read confirmation").WithReason(err).Done()
		}
		if !confirmed {
			fmt.Println("Aborted; no changes were made.")
			return
		}
	}

	err = d.configFactory.
		NewRequest().
		WithMethod(http.MethodDelete).
		WithPath("o/:organisation/users/" + userID).
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to delete user").WithReason(err).Done()
	}

	fmt.Printf("Deleted end-user %s.\n", label)
}

func (d *deleteUser) fetchUsers() []v1.EndUser {
	resp := v1.EndUserPage{}
	err := d.configFactory.
		NewRequest().
		WithMethod(http.MethodGet).
		Into(&resp).
		WithPath("o/:organisation/users").
		Do()
	if err != nil {
		utils.NewExitError().WithMessage("failed to list users").WithReason(err).Done()
	}
	return resp.Users
}

// resolveDeleteUserID accepts a user ULID or an external ID. ULIDs are returned as-is so a
// caller that already has the platform id (the VS Code delete button) does not round-trip the
// list. External IDs are matched against the org's end-user list.
func resolveDeleteUserID(value string, users []v1.EndUser) (string, error) {
	if _, err := ulid.Parse(value); err == nil {
		return value, nil
	}

	for _, u := range users {
		if u.ExternalID == value {
			return u.ID.String(), nil
		}
	}

	return "", fmt.Errorf("no end-user found with ULID or external ID matching %s", value)
}

func isInteractive() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
