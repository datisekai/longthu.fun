// Package shortcode generates random, unambiguous short codes for public
// URL surfaces (group share codes, player codes, payment intent codes).
//
// Alphabet: Crockford base32 — 32 characters excluding I, L, O, U.
// I/L look like 1; O looks like 0; U is excluded per Crockford's spec to
// reduce accidental obscenities in random output. The result is reliably
// hand-typable and unambiguous when read aloud.
//
// See architecture.md §FR-30 short-code unguessability and PRD §15 privacy.
package shortcode

import (
	"context"
	"crypto/rand"
	"fmt"
)

// alphabet is Crockford base32 (32 chars).
// Position-to-char only; we do not need decode (server-side compare-only).
const alphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// Kind documents which surface a code is for. It doesn't enforce length —
// callers pass length explicitly per FR-30's asymmetric requirements
// (Group ≥6, Player ≥8, PaymentIntent 6). The Kind exists for clarity in
// call sites and future extension (e.g. per-Kind rate-limit policies).
type Kind int

const (
	GroupShare Kind = iota
	PlayerCode
	PaymentIntent
)

const minLength = 4

// Generate returns a length-N random code from the Crockford alphabet.
// Returns an error if length < 4 (too easy to brute-force).
func Generate(length int, _ Kind) (string, error) {
	if length < minLength {
		return "", fmt.Errorf("shortcode.Generate: length %d below minimum %d", length, minLength)
	}

	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("shortcode.Generate: crypto/rand: %w", err)
	}

	out := make([]byte, length)
	for i, b := range buf {
		// 256 / 32 = 8 — even division, zero bias.
		out[i] = alphabet[b&0x1F]
	}
	return string(out), nil
}

// ExistsFunc is the collision-check callback passed to GenerateUnique.
// Implementations typically run a SELECT against the table that holds the
// code, returning (true, nil) when a duplicate exists.
type ExistsFunc func(ctx context.Context, code string) (bool, error)

const maxRetries = 10

// GenerateUnique loops Generate up to maxRetries times, calling `exists` to
// check the database after each generation. Returns the first code for
// which exists returns false. Returns an error if all attempts collide.
func GenerateUnique(ctx context.Context, length int, kind Kind, exists ExistsFunc) (string, error) {
	for attempt := 0; attempt < maxRetries; attempt++ {
		code, err := Generate(length, kind)
		if err != nil {
			return "", err
		}
		taken, err := exists(ctx, code)
		if err != nil {
			return "", fmt.Errorf("shortcode.GenerateUnique: exists check failed: %w", err)
		}
		if !taken {
			return code, nil
		}
	}
	return "", fmt.Errorf("shortcode.GenerateUnique: exhausted %d retries at length %d", maxRetries, length)
}
