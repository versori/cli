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

package ulid

import (
	"crypto/rand"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

// ULID is a wrapper around ulid.ULID which implements sql.Scanner that allows UUID-encoded strings.
type ULID struct {
	ulid.ULID
}

// New is a wrapper around ulid.New.
func New(ms uint64, entropy io.Reader) (id ULID, err error) {
	u, err := ulid.New(ms, entropy)

	return ULID{ULID: u}, err
}

// NewDefault is a helper function that just gives a ULID with no effort.
func NewDefault() (id ULID, err error) {
	u, err := ulid.New(ulid.Now(), rand.Reader)

	return ULID{ULID: u}, err
}

// MustDefault is a helper function that just gives a ULID with no effort.
func MustDefault() (id ULID) {
	u, err := ulid.New(ulid.Now(), rand.Reader)
	if err != nil {
		panic(err)
	}

	return ULID{ULID: u}
}

// MustNew is a wrapper around ulid.MustNew.
func MustNew(ms uint64, entropy io.Reader) ULID {
	return ULID{ULID: ulid.MustNew(ms, entropy)}
}

// Parse is a wrapper around ulid.Parse.
func Parse(value string) (ULID, error) {
	parsed, err := ulid.Parse(value)

	return ULID{ULID: parsed}, err
}

// ParseStrict is a wrapper around ulid.ParseStrict.
func ParseStrict(value string) (id ULID, err error) {
	u, err := ulid.ParseStrict(value)

	return ULID{ULID: u}, err
}

// MustParse is a wrapper around ulid.MustParse.
func MustParse(value string) ULID {
	return ULID{ULID: ulid.MustParse(value)}
}

// MustParseStrict is a wrapper around ulid.MustParseStrict.
func MustParseStrict(value string) ULID {
	return ULID{ULID: ulid.MustParseStrict(value)}
}

// MaxTime is a wrapper around ulid.MaxTime.
func MaxTime() uint64 { return ulid.MaxTime() }

// Now is a wrapper around ulid.Now.
func Now() uint64 { return Timestamp(time.Now().UTC()) }

// Timestamp is a wrapper around ulid.Timestamp.
func Timestamp(t time.Time) uint64 {
	return ulid.Timestamp(t)
}

// Time is a wrapper around ulid.Time.
func Time(ms uint64) time.Time {
	return ulid.Time(ms)
}

// Monotonic is a wrapper around ulid.Monotonic.
func Monotonic(entropy io.Reader, inc uint64) *ulid.MonotonicEntropy {
	return ulid.Monotonic(entropy, inc)
}

func (id *ULID) Scan(src interface{}) error {
	if str, ok := src.(string); ok {
		if len(str) != ulid.EncodedSize {
			bytes, err := uuid.Parse(str)
			if err != nil {
				return err
			}

			return id.UnmarshalBinary(bytes[:])
		}
	}

	return id.ULID.Scan(src)
}

func (id *ULID) IsZero() bool {
	return id == nil || *id == ULID{}
}
