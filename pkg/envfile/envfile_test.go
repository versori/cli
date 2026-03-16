package envfile

import (
	"os"
	"testing"
)

func TestResolve(t *testing.T) {
	store := &Store{vars: map[string]string{
		"API_KEY":    "secret-123",
		"EMPTY_VAR":  "",
		"TOKEN_URL":  "https://example.com/token",
	}}

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "literal value unchanged",
			input: "plain-value",
			want:  "plain-value",
		},
		{
			name:  "dollar var reference",
			input: "$API_KEY",
			want:  "secret-123",
		},
		{
			name:  "braced var reference",
			input: "${API_KEY}",
			want:  "secret-123",
		},
		{
			name:  "braced var with url value",
			input: "${TOKEN_URL}",
			want:  "https://example.com/token",
		},
		{
			name:    "missing var returns error",
			input:   "$DOES_NOT_EXIST",
			wantErr: true,
		},
		{
			name:    "missing braced var returns error",
			input:   "${DOES_NOT_EXIST}",
			wantErr: true,
		},
		{
			name:    "empty dollar reference",
			input:   "$",
			wantErr: true,
		},
		{
			name:    "empty braced reference",
			input:   "${}",
			wantErr: true,
		},
		{
			name:  "empty string literal",
			input: "",
			want:  "",
		},
		{
			name:  "value with dollar in middle is literal",
			input: "not$avar",
			want:  "not$avar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := store.Resolve(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Resolve(%q) expected error, got %q", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Errorf("Resolve(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("Resolve(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolve_FallbackToEnv(t *testing.T) {
	store := Empty()

	t.Setenv("FALLBACK_VAR", "from-env")

	got, err := store.Resolve("$FALLBACK_VAR")
	if err != nil {
		t.Fatalf("Resolve($FALLBACK_VAR) unexpected error: %v", err)
	}
	if got != "from-env" {
		t.Errorf("Resolve($FALLBACK_VAR) = %q, want %q", got, "from-env")
	}
}

func TestResolve_StoreTakesPrecedence(t *testing.T) {
	store := &Store{vars: map[string]string{
		"PRIORITY_VAR": "from-store",
	}}

	t.Setenv("PRIORITY_VAR", "from-env")

	got, err := store.Resolve("$PRIORITY_VAR")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "from-store" {
		t.Errorf("Resolve($PRIORITY_VAR) = %q, want %q (store should take precedence)", got, "from-store")
	}
}

func TestLoad_NonExistentFile(t *testing.T) {
	_, err := Load("/nonexistent/.env")
	if err == nil {
		t.Error("Load of nonexistent file should return error")
	}
}

func TestLoad_ValidFile(t *testing.T) {
	tmp, err := os.CreateTemp("", "envfile-test-*.env")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())

	_, err = tmp.WriteString("MY_KEY=hello\nMY_SECRET=world\n")
	if err != nil {
		t.Fatal(err)
	}
	tmp.Close()

	store, err := Load(tmp.Name())
	if err != nil {
		t.Fatalf("Load(%q) unexpected error: %v", tmp.Name(), err)
	}

	got, err := store.Resolve("$MY_KEY")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello" {
		t.Errorf("Resolve($MY_KEY) = %q, want %q", got, "hello")
	}

	got, err = store.Resolve("${MY_SECRET}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "world" {
		t.Errorf("Resolve(${MY_SECRET}) = %q, want %q", got, "world")
	}
}

func TestEmpty(t *testing.T) {
	store := Empty()
	if store.vars == nil {
		t.Error("Empty() should return store with non-nil map")
	}

	got, err := store.Resolve("literal")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "literal" {
		t.Errorf("Resolve(%q) = %q, want %q", "literal", got, "literal")
	}
}
