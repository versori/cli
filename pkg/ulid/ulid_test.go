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
	"testing"

	"github.com/oklog/ulid/v2"
)

func TestULID_Scan(t *testing.T) {
	type args struct {
		src interface{}
	}
	tests := []struct {
		name     string
		args     args
		wantErr  bool
		wantULID ULID
	}{
		{
			name: "parses ULIDs from UUID string format",
			args: args{
				src: "017cf063-8a5e-0000-0000-000000000000",
			},
			wantULID: ULID{ulid.MustParse("01FKR672JY0000000000000000")},
		},
		{
			name: "parses ULIDs from ULID string format",
			args: args{
				src: "01FKR672JY0000000000000000",
			},
			wantULID: ULID{ulid.MustParse("01FKR672JY0000000000000000")},
		},
		{
			name: "parses ULIDs from []byte format",
			args: args{
				src: []byte{1, 124, 240, 99, 138, 94, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			},
			wantULID: ULID{ulid.MustParse("01FKR672JY0000000000000000")},
		},
		{
			name: "errors on invalid ULID",
			args: args{
				src: "zzzzzzzz-zzzz-zzzz-zzzz-zzzz",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var id ULID
			if err := id.Scan(tt.args.src); (err != nil) != tt.wantErr {
				t.Errorf("Scan() error = %v, wantErr %v", err, tt.wantErr)
			}

			if id != tt.wantULID {
				t.Errorf("ULID: want = %s, got = %s", tt.wantULID.String(), id.String())
			}
		})
	}
}

func TestULID_IsZero(t *testing.T) {
	tests := []struct {
		name  string
		input *ULID
		want  bool
	}{
		{
			name:  "nil is zero",
			input: nil,
			want:  true,
		},
		{
			name:  "zero value is zero",
			input: &ULID{},
			want:  true,
		},
		{
			name:  "defined value is not zero",
			input: &ULID{ULID: [16]byte{1, 124, 240, 99, 138, 94, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
			want:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.input.IsZero(); got != tt.want {
				t.Errorf("IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}
