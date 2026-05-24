// Package money provides VND formatting + parsing helpers.
//
// Mirrors frontend/src/lib/money.ts in behavior — keep them in sync when
// either side changes (Story 1.3 establishes the contract).
//
// VND has no fractional unit; all amounts are integer values in `đồng`
// (the smallest unit). See architecture.md §Implementation Patterns
// "JSON & data formats — Money".
package money

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const (
	thousand int64 = 1_000
	million  int64 = 1_000_000
)

// Mode controls how Format renders an amount.
type Mode int

const (
	// Full renders with dot-separated thousands and a `đ` suffix (Vietnamese norm).
	Full Mode = iota
	// Compact renders bucketed to k / tr with one decimal, no trailing `.0`.
	Compact
)

// Format an amount as a Vietnamese-norm money string.
//
// See Mode for behavior. Negative amounts are formatted with a leading `-`;
// the app does not normally display negatives.
func Format(amount int64, mode Mode) string {
	sign := ""
	abs := amount
	if abs < 0 {
		sign = "-"
		abs = -abs
	}

	if mode == Compact {
		switch {
		case abs >= million:
			return sign + compactUnit(float64(abs)/float64(million), "tr")
		case abs >= thousand:
			return sign + compactUnit(float64(abs)/float64(thousand), "k")
		default:
			return sign + strconv.FormatInt(abs, 10)
		}
	}

	// Full mode — insert a `.` every 3 digits from the right.
	digits := strconv.FormatInt(abs, 10)
	n := len(digits)
	var b strings.Builder
	for i, r := range digits {
		if i > 0 && (n-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(r)
	}
	b.WriteString("đ")
	return sign + b.String()
}

// compactUnit formats a float as "<value><suffix>" with one decimal max
// and no trailing `.0`.
func compactUnit(value float64, suffix string) string {
	// Truncate (not round) to one decimal — matches the TS Math.floor behavior.
	truncated := float64(int64(value*10)) / 10
	text := strconv.FormatFloat(truncated, 'f', 1, 64)
	text = strings.TrimSuffix(text, ".0")
	return text + suffix
}

var (
	// Trailing currency markers we strip during parse.
	currencySuffixRe = regexp.MustCompile(`(?i)\s*(đồng|vnđ|vnd|đ)\s*$`)
	// Compact `triệu` / `tr` forms.
	compactMillionRe = regexp.MustCompile(`^([\d.,]+)\s*(triệu|tr)$`)
	// Compact `nghìn` / `nghin` / `k` forms.
	compactThousandRe = regexp.MustCompile(`^([\d.,]+)\s*(nghìn|nghin|k)$`)
	// Pure numeric (with optional dots / commas as thousands separators).
	plainNumericRe = regexp.MustCompile(`^[\d.,]+$`)
)

// Parse a Vietnamese-form money string back to its integer VND value.
//
// See ParseExamples in the package-level docs for what inputs are accepted.
// Returns an error on empty, garbage, or negative input.
func Parse(input string) (int64, error) {
	s := strings.ToLower(strings.TrimSpace(input))
	if s == "" {
		return 0, fmt.Errorf("money.Parse: empty input")
	}

	// Strip currency suffix.
	s = currencySuffixRe.ReplaceAllString(s, "")
	s = strings.TrimSpace(s)

	// Compact triệu / tr — check before nghìn / k.
	if m := compactMillionRe.FindStringSubmatch(s); m != nil {
		f, err := parseDecimal(m[1])
		if err != nil {
			return 0, fmt.Errorf("money.Parse: not a number %q", input)
		}
		return rejectNegative(int64(f*float64(million)+0.5), input)
	}

	// Compact nghìn / k.
	if m := compactThousandRe.FindStringSubmatch(s); m != nil {
		f, err := parseDecimal(m[1])
		if err != nil {
			return 0, fmt.Errorf("money.Parse: not a number %q", input)
		}
		return rejectNegative(int64(f*float64(thousand)+0.5), input)
	}

	// Plain number with thousands separators.
	if plainNumericRe.MatchString(s) {
		stripped := strings.ReplaceAll(strings.ReplaceAll(s, ".", ""), ",", "")
		n, err := strconv.ParseInt(stripped, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("money.Parse: not a number %q", input)
		}
		return rejectNegative(n, input)
	}

	return 0, fmt.Errorf("money.Parse: cannot parse %q", input)
}

// parseDecimal accepts "1.5" or "1,5" as 1.5.
func parseDecimal(s string) (float64, error) {
	s = strings.ReplaceAll(s, ",", ".")
	return strconv.ParseFloat(s, 64)
}

func rejectNegative(n int64, original string) (int64, error) {
	if n < 0 {
		return 0, fmt.Errorf("money.Parse: negative not allowed %q", original)
	}
	return n, nil
}

// Spoken renders an amount as Vietnamese spoken form for screen-reader
// `aria-label`. Matches the COMPACT display's precision (not the underlying
// value's full digits) so the audio aligns with the visible tile.
//
// Examples:
//
//	Spoken(120_000)   == "120 nghìn đồng"
//	Spoken(1_234_567) == "1 triệu 200 nghìn đồng"  (rounds to 1.2tr precision)
//	Spoken(2_000_000) == "2 triệu đồng"
//	Spoken(0)         == "0 đồng"
func Spoken(amount int64) string {
	if amount < 0 {
		return ""
	}
	if amount == 0 {
		return "0 đồng"
	}

	if amount >= million {
		tenths := amount / 100_000   // 12 for 1234567
		wholeMillions := tenths / 10 // 1
		tenthDigit := tenths % 10    // 2
		parts := []string{fmt.Sprintf("%d triệu", wholeMillions)}
		if tenthDigit > 0 {
			parts = append(parts, fmt.Sprintf("%d nghìn", tenthDigit*100))
		}
		return strings.Join(parts, " ") + " đồng"
	}

	if amount >= thousand {
		// Round to nearest thousand (matches the "k" compact display).
		k := (amount + 500) / thousand
		return fmt.Sprintf("%d nghìn đồng", k)
	}

	return fmt.Sprintf("%d đồng", amount)
}
