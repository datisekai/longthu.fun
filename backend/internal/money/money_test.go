package money

import (
	"strings"
	"testing"
)

func TestFormat_Full(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0đ"},
		{1_000, "1.000đ"},
		{120_000, "120.000đ"},
		{1_234_567, "1.234.567đ"},
		{-120_000, "-120.000đ"},
	}
	for _, c := range cases {
		got := Format(c.in, Full)
		if got != c.want {
			t.Errorf("Format(%d, Full) = %q; want %q", c.in, got, c.want)
		}
	}
}

func TestFormat_Compact(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{500, "500"},
		{1_000, "1k"},
		{1_200, "1.2k"},
		{120_000, "120k"},
		{1_000_000, "1tr"},
		{1_200_000, "1.2tr"},
		{2_000_000, "2tr"},
		{1_234_567, "1.2tr"},
	}
	for _, c := range cases {
		got := Format(c.in, Compact)
		if got != c.want {
			t.Errorf("Format(%d, Compact) = %q; want %q", c.in, got, c.want)
		}
	}
}

func TestParse_Accepts(t *testing.T) {
	cases := []struct {
		in   string
		want int64
	}{
		// Plain integers.
		{"120000", 120_000},
		{"0", 0},

		// Dot-thousands (Vietnamese norm).
		{"120.000", 120_000},
		{"1.234.567", 1_234_567},

		// Comma-thousands (US convention).
		{"120,000", 120_000},

		// Compact k / nghìn.
		{"120k", 120_000},
		{"120 nghìn", 120_000},
		{"120nghin", 120_000},
		{"1.5k", 1_500},

		// Compact tr / triệu.
		{"1tr", 1_000_000},
		{"1.5tr", 1_500_000},
		{"1.5 triệu", 1_500_000},
		{"1.2tr", 1_200_000},

		// With currency suffix.
		{"120.000đ", 120_000},
		{"120 nghìn đồng", 120_000},
		{"120000 vnđ", 120_000},

		// Whitespace + case.
		{"  120 K  ", 120_000},
		{"1.2 TR", 1_200_000},
	}
	for _, c := range cases {
		got, err := Parse(c.in)
		if err != nil {
			t.Errorf("Parse(%q) unexpected error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("Parse(%q) = %d; want %d", c.in, got, c.want)
		}
	}
}

func TestParse_Rejects(t *testing.T) {
	cases := []string{"", "   ", "abc", "-1000"}
	for _, in := range cases {
		_, err := Parse(in)
		if err == nil {
			t.Errorf("Parse(%q) expected error, got nil", in)
		}
	}
}

func TestRoundTrip_Compact(t *testing.T) {
	// Compact preserves exactly for round amounts.
	for _, n := range []int64{1_000, 120_000, 1_000_000, 1_200_000, 2_000_000} {
		formatted := Format(n, Compact)
		got, err := Parse(formatted)
		if err != nil {
			t.Errorf("Parse(Format(%d, Compact)) = error %v", n, err)
			continue
		}
		if got != n {
			t.Errorf("RoundTrip(%d) = %d; want %d", n, got, n)
		}
	}

	// 1_234_567 rounds to 1.2tr (compact is lossy by design).
	formatted := Format(1_234_567, Compact)
	if formatted != "1.2tr" {
		t.Errorf("Format(1234567, Compact) = %q; want %q", formatted, "1.2tr")
	}
	got, err := Parse(formatted)
	if err != nil {
		t.Fatalf("Parse(%q) error: %v", formatted, err)
	}
	if got != 1_200_000 {
		t.Errorf("Parse(%q) = %d; want %d", formatted, got, 1_200_000)
	}
}

func TestRoundTrip_Full(t *testing.T) {
	for _, n := range []int64{0, 1_000, 120_000, 1_234_567, 999_999_999} {
		formatted := Format(n, Full)
		got, err := Parse(formatted)
		if err != nil {
			t.Errorf("Parse(Format(%d, Full)) = error %v", n, err)
			continue
		}
		if got != n {
			t.Errorf("RoundTrip(%d) = %d; want %d", n, got, n)
		}
	}
}

func TestSpoken(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0 đồng"},
		{500, "500 đồng"},
		{999, "999 đồng"},
		{120_000, "120 nghìn đồng"},
		{12_500, "13 nghìn đồng"}, // rounds nearest
		{12_499, "12 nghìn đồng"},
		{1_000_000, "1 triệu đồng"},
		{2_000_000, "2 triệu đồng"},
		{1_500_000, "1 triệu 500 nghìn đồng"},
		{2_500_000, "2 triệu 500 nghìn đồng"},
		{1_234_567, "1 triệu 200 nghìn đồng"}, // rounds to 1.2tr precision
		{1_200_000, "1 triệu 200 nghìn đồng"},
	}
	for _, c := range cases {
		got := Spoken(c.in)
		if got != c.want {
			t.Errorf("Spoken(%d) = %q; want %q", c.in, got, c.want)
		}
	}

	// Negative returns empty string.
	if got := Spoken(-1); got != "" {
		t.Errorf("Spoken(-1) = %q; want empty string", got)
	}
}

func TestSpoken_ContainsUnits(t *testing.T) {
	// Sanity: every non-zero output ends with " đồng".
	for _, n := range []int64{500, 1_000, 120_000, 1_500_000} {
		got := Spoken(n)
		if !strings.HasSuffix(got, " đồng") {
			t.Errorf("Spoken(%d) = %q; expected trailing ' đồng'", n, got)
		}
	}
}
