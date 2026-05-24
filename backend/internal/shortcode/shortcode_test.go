package shortcode

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestGenerate_Length(t *testing.T) {
	for _, n := range []int{4, 6, 8, 12, 24} {
		code, err := Generate(n, GroupShare)
		if err != nil {
			t.Fatalf("Generate(%d) error: %v", n, err)
		}
		if len(code) != n {
			t.Errorf("Generate(%d) len=%d; want %d", n, len(code), n)
		}
	}
}

func TestGenerate_LengthValidation(t *testing.T) {
	for _, n := range []int{0, 1, 2, 3, -1} {
		_, err := Generate(n, GroupShare)
		if err == nil {
			t.Errorf("Generate(%d) expected error, got nil", n)
		}
	}
}

func TestGenerate_Alphabet(t *testing.T) {
	const samples = 1000
	for i := 0; i < samples; i++ {
		code, err := Generate(8, GroupShare)
		if err != nil {
			t.Fatalf("Generate error: %v", err)
		}
		for _, c := range code {
			if !strings.ContainsRune(alphabet, c) {
				t.Fatalf("Generate produced char %q not in Crockford alphabet (code=%q)", c, code)
			}
		}
		// Explicit anti-checks: must not contain I, L, O, U.
		if strings.ContainsAny(code, "ILOU") {
			t.Fatalf("Generate produced excluded char in %q", code)
		}
	}
}

func TestGenerate_NoDuplicatesIn10k(t *testing.T) {
	const samples = 10_000
	seen := make(map[string]struct{}, samples)
	for i := 0; i < samples; i++ {
		code, err := Generate(6, GroupShare)
		if err != nil {
			t.Fatalf("Generate error at iter %d: %v", i, err)
		}
		if _, dup := seen[code]; dup {
			t.Fatalf("duplicate %q at iteration %d (P(this) ≈ 10⁻⁴ — investigate RNG)", code, i)
		}
		seen[code] = struct{}{}
	}
}

func TestGenerateUnique_RetryUntilFree(t *testing.T) {
	calls := 0
	exists := func(_ context.Context, _ string) (bool, error) {
		calls++
		return calls < 4, nil // true for calls 1, 2, 3; false on 4
	}
	code, err := GenerateUnique(context.Background(), 6, GroupShare, exists)
	if err != nil {
		t.Fatalf("GenerateUnique error: %v", err)
	}
	if code == "" {
		t.Fatal("GenerateUnique returned empty code")
	}
	if calls != 4 {
		t.Errorf("exists called %d times; want 4", calls)
	}
}

func TestGenerateUnique_ExhaustsRetries(t *testing.T) {
	exists := func(_ context.Context, _ string) (bool, error) {
		return true, nil // always collides
	}
	_, err := GenerateUnique(context.Background(), 6, GroupShare, exists)
	if err == nil {
		t.Fatal("GenerateUnique expected error after exhausting retries, got nil")
	}
	if !strings.Contains(err.Error(), "exhausted") {
		t.Errorf("error message %q does not mention exhausted retries", err.Error())
	}
}

func TestGenerateUnique_ExistsErrorPropagates(t *testing.T) {
	wantErr := errors.New("simulated DB error")
	exists := func(_ context.Context, _ string) (bool, error) {
		return false, wantErr
	}
	_, err := GenerateUnique(context.Background(), 6, GroupShare, exists)
	if err == nil || !errors.Is(err, wantErr) {
		t.Errorf("GenerateUnique should wrap exists error; got %v", err)
	}
}
