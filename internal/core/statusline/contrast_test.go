package statusline

import (
	"math"
	"testing"

	"github.com/intentdriven/abcd/internal/core/mode"
)

// TestParseColor pins the accepted spellings of a colour and the refusal of
// everything else. A setting file is a trust-boundary read, so a malformed
// value is an error the reader reports, never a silently substituted black.
func TestParseColor(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    RGB
		wantErr bool
	}{
		{name: "six digits", in: "#ffd700", want: RGB{0xff, 0xd7, 0x00}},
		{name: "six digits upper", in: "#FFD700", want: RGB{0xff, 0xd7, 0x00}},
		{name: "three digits expands", in: "#fd0", want: RGB{0xff, 0xdd, 0x00}},
		{name: "no hash", in: "ffd700", wantErr: true},
		{name: "empty", in: "", wantErr: true},
		{name: "short", in: "#ff", wantErr: true},
		{name: "long", in: "#ffd7000", wantErr: true},
		{name: "not hex", in: "#gggggg", wantErr: true},
		{name: "named colour", in: "gold", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseColor(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ParseColor(%q) = %v, want an error", tc.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseColor(%q): %v", tc.in, err)
			}
			if got != tc.want {
				t.Fatalf("ParseColor(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

// TestContrastMatchesTheRecordedMeasurements is the load-bearing arithmetic
// test: the three pairs iss-168 measured on 2026-08-29 against the same dark
// grey, quoted in the record to two decimals. If the relative-luminance
// implementation drifts, these three numbers move.
func TestContrastMatchesTheRecordedMeasurements(t *testing.T) {
	cases := []struct {
		name   string
		fg, bg string
		want   float64
	}{
		// iss-168 measured #ffd700; the shipped pair is livery's house yellow,
		// chosen over the recorded hex on the maintainer's ruling of 2026-09-15
		// so the badge matches abcd's visual identity. The record's reasoning was
		// about HUE -- a gold between white's quiet presence and yellow's caution
		// -- and that survives the swap; only the measured ratio moved.
		{name: "the shipped house yellow", fg: "#f0c052", bg: "#444444", want: 5.74},
		{name: "the gold iss-168 measured", fg: "#ffd700", bg: "#444444", want: 6.94},
		{name: "the white it was chosen against", fg: "#ffffff", bg: "#444444", want: 9.74},
		{name: "the pure yellow it was chosen against", fg: "#ffff00", bg: "#444444", want: 9.07},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Contrast(tc.fg, tc.bg)
			if err != nil {
				t.Fatalf("Contrast(%q,%q): %v", tc.fg, tc.bg, err)
			}
			if math.Abs(round2(got)-tc.want) > 0.0001 {
				t.Fatalf("Contrast(%q,%q) = %.4f (%.2f), want %.2f", tc.fg, tc.bg, got, round2(got), tc.want)
			}
		})
	}
}

// TestContrastIsSymmetricAndBounded pins the two properties of the WCAG ratio
// that callers rely on: order does not matter, and the extremes are 1 and 21.
func TestContrastIsSymmetricAndBounded(t *testing.T) {
	same, err := Contrast("#808080", "#808080")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(same-1) > 0.0001 {
		t.Fatalf("a colour against itself = %.4f, want 1", same)
	}
	black, err := Contrast("#000000", "#ffffff")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(black-21) > 0.0001 {
		t.Fatalf("black on white = %.4f, want 21", black)
	}
	white, err := Contrast("#ffffff", "#000000")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(white-black) > 1e-12 {
		t.Fatalf("Contrast is not symmetric: %.6f vs %.6f", white, black)
	}
}

// TestContrastBarAdmitsTheDefault is the constraint that fixes the bar. The
// presence default measures 5.74:1, so a bar of 7:1 (WCAG AAA for body text)
// would refuse the record's own settled default -- and so would 6:1, which the
// earlier #ffd700 default would have cleared. The margin narrowed when the
// palette won the 2026-09-15 ruling; the bar itself did not move. 4.5:1 (AA, normal text) is
// the bar, and this test is what stops it being raised by accident.
func TestContrastBarAdmitsTheDefault(t *testing.T) {
	if ContrastBar != 4.5 {
		t.Fatalf("ContrastBar = %v, want 4.5 (WCAG 2.2 AA, normal text)", ContrastBar)
	}
	d := Defaults()
	got, err := Contrast(d.Presence.Foreground, d.Presence.Background)
	if err != nil {
		t.Fatal(err)
	}
	if got < ContrastBar {
		t.Fatalf("the shipped default measures %.2f:1, below its own bar of %v:1", got, ContrastBar)
	}
}

// TestRoleBadgePairsClearTheBar holds the fixed role pairs to the same bar the
// configured presence pair is held to (spc-70, "the role badges' pairs are
// fixed constants held to the same bar").
func TestRoleBadgePairsClearTheBar(t *testing.T) {
	for _, st := range mode.States() {
		pair := fixedPair(st)
		if st == StateManaged {
			pair = Defaults().Presence
		}
		got, err := Contrast(pair.Foreground, pair.Background)
		if err != nil {
			t.Fatalf("%s: %v", st, err)
		}
		if got < ContrastBar {
			t.Fatalf("%s badge measures %.2f:1, below the %v:1 bar", st, got, ContrastBar)
		}
	}
}

// TestFormatRatio pins the wording a refusal reports the measurement in.
func TestFormatRatio(t *testing.T) {
	if got := FormatRatio(6.9441); got != "6.94:1" {
		t.Fatalf("FormatRatio(6.9441) = %q, want %q", got, "6.94:1")
	}
	if got := FormatRatio(1); got != "1.00:1" {
		t.Fatalf("FormatRatio(1) = %q, want %q", got, "1.00:1")
	}
}

func round2(f float64) float64 { return math.Round(f*100) / 100 }
