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

package envfile

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Store holds key-value pairs loaded from a .env file.
type Store struct {
	vars map[string]string
}

// Load reads a .env file and returns a Store with its contents.
func Load(path string) (*Store, error) {
	vars, err := godotenv.Read(path)
	if err != nil {
		return nil, fmt.Errorf("reading env file %s: %w", path, err)
	}

	return &Store{vars: vars}, nil
}

// Empty returns a Store with no variables.
func Empty() *Store {
	return &Store{vars: map[string]string{}}
}

// Resolve resolves a value that may be a variable reference ($VAR or ${VAR}).
// If the value is a reference, it looks up the variable in the store first,
// then falls back to os.Getenv. Non-reference values are returned unchanged.
// Returns an error if a referenced variable is not found anywhere.
func (s *Store) Resolve(value string) (string, error) {
	name := ""

	switch {
	case strings.HasPrefix(value, "${") && strings.HasSuffix(value, "}"):
		name = value[2 : len(value)-1]
	case strings.HasPrefix(value, "$"):
		name = value[1:]
	default:
		return value, nil
	}

	if name == "" {
		return "", fmt.Errorf("empty variable reference: %s", value)
	}

	if v, ok := s.vars[name]; ok {
		return v, nil
	}

	if v := os.Getenv(name); v != "" {
		return v, nil
	}

	return "", fmt.Errorf("variable %q not found in env file or environment", name)
}
