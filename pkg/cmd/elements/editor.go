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

type EditorOpts func(*Editor)

// WithValidation sets a validation function for the Editor element. The validation function
// takes the input string and returns an error if the input is invalid, or nil if it is valid.
func WithValidation(validate func(string) error) EditorOpts {
	return func(e *Editor) {
		e.validate = validate
	}
}

type Editor struct {
	title     string
	mutliLine bool
	validate  func(string) error
}

// NewEditor creates a new Editor element with the given title. The title is only rendered if
// it is not an empty string. The multiLine parameter determines whether the editor
// allows multiple lines of input or just a single line.
func NewEditor(title string, multiLine bool, opts ...EditorOpts) *Editor {
	e := &Editor{
		title:     title,
		mutliLine: multiLine,
	}

	for _, opt := range opts {
		opt(e)
	}

	return e
}

func (e *Editor) Edit(content *string) error {
	if e.mutliLine {
		return e.editMultiLineText(content)
	}

	return e.editText(content)
}

func (e *Editor) editText(content *string) error {
	input := huh.NewInput()

	if e.title != "" {
		input = input.Title(e.title)
	}

	if e.validate != nil {
		input = input.Validate(e.validate)
	}

	field := input.Prompt(">").Value(content)

	return huh.NewForm(
		huh.NewGroup(field),
	).Run()
}

func (e *Editor) editMultiLineText(content *string) error {
	input := huh.NewText().Value(content).CharLimit(2000)

	if e.title != "" {
		input = input.Title(e.title)
	}

	if e.validate != nil {
		input = input.Validate(e.validate)
	}

	return huh.NewForm(
		huh.NewGroup(input),
	).Run()
}
