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

package elements

import (
	"github.com/charmbracelet/huh"
)

type Confirm struct {
	title       string
	affirmative string
	negative    string
}

// NewConfirm creates a new Y/N Confirm element with the given title. Empty affirmative/negative
// strings fall back to "Yes" / "No" so most callers can just pass the title.
func NewConfirm(title string) *Confirm {
	return &Confirm{
		title:       title,
		affirmative: "Yes",
		negative:    "No",
	}
}

// WithLabels overrides the Yes/No button labels (e.g. for "Required (Y/N)" → "Required" / "Optional").
func (c *Confirm) WithLabels(affirmative, negative string) *Confirm {
	c.affirmative = affirmative
	c.negative = negative

	return c
}

// Confirm renders the confirmation prompt and binds the user's choice to the provided pointer.
func (c *Confirm) Confirm(value *bool) error {
	confirm := huh.NewConfirm().
		Affirmative(c.affirmative).
		Negative(c.negative).
		Value(value)

	if c.title != "" {
		confirm = confirm.Title(c.title)
	}

	return huh.NewForm(
		huh.NewGroup(confirm),
	).Run()
}
