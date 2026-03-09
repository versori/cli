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

import "github.com/charmbracelet/huh"

type ListSelect struct {
	title   string
	options map[string]string
}

// NewListSelect creates a new ListSelect element with the given title. The title is only rendered if
// it is not an empty string
func NewListSelect(title string) *ListSelect {
	return &ListSelect{
		title:   title,
		options: make(map[string]string),
	}
}

// AddOption adds an option to the ListSelect element. The label is the text displayed to the user,
// and the value is the value that will be returned when the option is selected.
// The value can be an empty string, in which case the label will be used as the value.
// Options are a map, so labels must be unique otherwise they will be overwritten.
func (ls *ListSelect) AddOption(label string, value string) {
	ls.options[label] = value
}

// Select renders the ListSelect element and binds the selected value to the provided pointer.
func (ls *ListSelect) Select(value *string) error {
	listselect := huh.NewSelect[string]()

	if ls.title != "" {
		listselect = listselect.Title(ls.title)
	}

	opts := make([]huh.Option[string], 0, len(ls.options))

	for k, v := range ls.options {
		var opt huh.Option[string]
		if v != "" {
			opt = huh.NewOption(k, v)
		} else {
			opt = huh.NewOption(k, k)
		}
		opts = append(opts, opt)
	}

	listselect.Options(opts...)

	listselect.Value(value)

	return huh.NewForm(
		huh.NewGroup(listselect),
	).Run()
}
