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

func Ptr[T any](v T) *T {
	return &v
}

// StringOrNil returns a pointer to the string if it's not empty, otherwise it returns nil.
func StringOrNil(s string) *string {
	if s == "" {
		return nil
	}

	return &s
}
