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

package utils

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

type ExitError struct {
	ExitCode int
	Message  string
	Reason   error
}

func NewExitError() *ExitError {
	return &ExitError{
		ExitCode: 1,
	}
}

// WithCode sets the exit code for the error
func (e *ExitError) WithCode(code int) *ExitError {
	e.ExitCode = code

	return e
}

// WithMessage sets the message for the error
func (e *ExitError) WithMessage(message string) *ExitError {
	e.Message = message

	return e
}

// WithReason sets the reason for the error
func (e *ExitError) WithReason(reason error) *ExitError {
	e.Reason = reason

	return e
}

// Done is a convenience method to exit the application with the error
func (e *ExitError) Done() {
	CheckError(e)
}

func (e *ExitError) Error() string {
	return e.Message
}

func (e *ExitError) Code() int {
	return e.ExitCode
}

// CheckError handles the error nad calls os.Exit
// If the error is not an ExitError, it is printed to stderr and the application exits with code 1
// If the error is an APIError, it is converted to an ExitError and handled as such
// If the error is nil, nothing happens
func CheckError(err error) {
	if err == nil {
		return
	}

	if exitErr, ok := errors.AsType[*ExitError](err); ok {
		if exitErr.Code() == 0 {
			os.Exit(0)
		}

		messageBuilder := strings.Builder{}

		if exitErr.Message != "" {
			messageBuilder.WriteString("Message: \n  " + strings.ReplaceAll(exitErr.Message, "\n", "\n  "))
			if !strings.HasSuffix(messageBuilder.String(), "\n") {
				messageBuilder.WriteString("\n")
			}
		}

		if exitErr.Reason != nil {
			// I want the message to look like this:
			// Message:
			//   something went wrong
			// Reason:
			//   something else went wrong
			if len(messageBuilder.String()) > 1 && !strings.HasSuffix(messageBuilder.String(), "\n") {
				messageBuilder.WriteString("\n")
			}

			messageBuilder.WriteString("Reason:\n  ")

			if apiReasonErr, ok2 := errors.AsType[*APIError](exitErr.Reason); ok2 {
				fmt.Fprintf(&messageBuilder, "API Message: %s\n", apiReasonErr.Message)
				fmt.Fprintf(&messageBuilder, "  API Details: %s\n", apiReasonErr.Details)
			} else {
				messageBuilder.WriteString(strings.ReplaceAll(exitErr.Reason.Error(), "\n", "\n  "))
			}

			if !strings.HasSuffix(messageBuilder.String(), "\n") {
				messageBuilder.WriteString("\n")
			}
		}

		fmt.Fprint(os.Stderr, messageBuilder.String())
		os.Exit(exitErr.Code())
	}

	if apiErr, ok := errors.AsType[*APIError](err); ok {
		// How efficient
		// If anyone asks we blame AI
		CheckError(NewExitError().WithMessage(apiErr.Message).WithReason(apiErr))
	}

	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

type APIError struct {
	Message string `json:"message"`
	Details string `json:"details"`
}

func (a *APIError) Error() string {
	return fmt.Sprintf("%s: %s", a.Message, a.Details)
}
