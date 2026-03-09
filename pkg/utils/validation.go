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
	"net/url"

	"github.com/go-jose/go-jose/v3/jwt"
	"github.com/versori/cli/pkg/ulid"
)

// IsValidURL checks if the input string is a valid URL.
func IsValidURL(input string) error {
	_, err := url.ParseRequestURI(input)
	if err != nil {
		return errors.New("invalid URL")
	}

	return nil
}

// IsValidJWT checks if the input string is a valid JWT token. Note that
// this function only checks the structure of the token and does not verify
// its signature or claims.
func IsValidJWT(input string) error {
	_, err := jwt.ParseSigned(input)
	if err != nil {
		return errors.New("invalid JWT token")
	}

	return nil
}

// IsValidULID checks if the input string is a valid ULID.
func IsValidULID(input string) error {
	_, err := ulid.Parse(input)
	if err != nil {
		return errors.New("invalid ULID")
	}

	return nil
}
