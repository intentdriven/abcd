package statusline

// Colour, and the one measurement that decides whether a configured pair is
// allowed to render.
//
// The bar is WCAG 2.2 success criterion 1.4.3 (Contrast (Minimum)) at level AA
// for normal-size text: 4.5:1. Three things fix it there rather than higher or
// lower.
//
// The badge is a short word at the host's own status-line size — normal text by
// the criterion's own definition (below 18pt / 14pt bold), so the 3:1
// large-text allowance does not apply and would be the wrong bar to borrow.
//
// The level-AAA bar (1.4.6, Contrast (Enhanced)) is 7:1, and the default this
// package ships measures 5.74:1 — the house yellow #f0c052 on #444444, the
// pair iss-168's colour ruling of 2026-09-15 settled when the palette won over
// the record's gold. The ruling kept the hue the record argued for; only the
// measurement moved: iss-168 had chosen on 2026-08-29 between gold at 6.94,
// pure yellow at 9.07 and white at 9.74, all against the same dark grey. A 7:1
// bar would refuse abcd's own settled default, and so would 6:1, which the
// earlier gold would have cleared — which is the proof that AAA is not the bar
// this design was drawn to. TestContrastBarAdmitsTheDefault holds that
// argument as a test.
//
// The ratio is computed from the WCAG relative-luminance formula verbatim
// ((L1 + 0.05) / (L2 + 0.05) over the sRGB linearisation), not from a
// perceptual model such as APCA. APCA is the better predictor and is where
// WCAG 3 is heading, but it is a working draft with no settled thresholds, and
// the record's measurements are WCAG 2 numbers. Matching the measurements the
// design was chosen by matters more here than using the better formula.

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// ContrastBar is the minimum contrast ratio a presence pair must measure to be
// allowed to render. See the package comment above for why it sits here.
const ContrastBar = 4.5

// RGB is one 8-bit-per-channel colour.
type RGB struct{ R, G, B uint8 }

// ParseColor reads a hex colour in the two spellings a settings file may use:
// "#rrggbb" and the "#rgb" shorthand, either case. Everything else is refused
// rather than coerced — a value the reader cannot understand is a setting the
// user wrote and did not get, and substituting black for it would hide that.
func ParseColor(s string) (RGB, error) {
	raw := strings.TrimSpace(s)
	if !strings.HasPrefix(raw, "#") {
		return RGB{}, fmt.Errorf("colour %q must be a hex value beginning with '#'", s)
	}
	digits := raw[1:]
	switch len(digits) {
	case 3:
		// "#fd0" means "#ffdd00": each digit is doubled, not zero-padded.
		var expanded strings.Builder
		for _, r := range digits {
			expanded.WriteRune(r)
			expanded.WriteRune(r)
		}
		digits = expanded.String()
	case 6:
	default:
		return RGB{}, fmt.Errorf("colour %q must carry 3 or 6 hex digits after the '#'", s)
	}
	v, err := strconv.ParseUint(digits, 16, 32)
	if err != nil {
		return RGB{}, fmt.Errorf("colour %q is not hexadecimal", s)
	}
	return RGB{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v)}, nil
}

// Luminance is the WCAG relative luminance of a colour, 0 for black and 1 for
// white.
func (c RGB) Luminance() float64 {
	return 0.2126*linearize(float64(c.R)/255) +
		0.7152*linearize(float64(c.G)/255) +
		0.0722*linearize(float64(c.B)/255)
}

// linearize undoes the sRGB transfer function for one channel. The 0.04045
// knee is the current (WCAG 2.1/2.2) value; the 2.0 draft's 0.03928 differs
// only in the last two decimal places of a channel below 4% and moves no ratio
// this package reports at two decimals.
func linearize(c float64) float64 {
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

// ContrastRGB is the WCAG contrast ratio of two colours: 1 for a colour
// against itself, 21 for black against white. It is symmetric — the lighter of
// the two is always the numerator — so a caller cannot get a different answer
// by naming the pair the other way round.
func ContrastRGB(a, b RGB) float64 {
	l1, l2 := a.Luminance(), b.Luminance()
	if l1 < l2 {
		l1, l2 = l2, l1
	}
	return (l1 + 0.05) / (l2 + 0.05)
}

// Contrast is ContrastRGB over two hex spellings, refusing either if it does
// not parse.
func Contrast(fg, bg string) (float64, error) {
	f, err := ParseColor(fg)
	if err != nil {
		return 0, err
	}
	b, err := ParseColor(bg)
	if err != nil {
		return 0, err
	}
	return ContrastRGB(f, b), nil
}

// FormatRatio renders a measured ratio the way a refusal reports it, to the
// two decimal places the record states its own measurements in ("6.94 to 1").
func FormatRatio(r float64) string { return strconv.FormatFloat(r, 'f', 2, 64) + ":1" }

// sgr renders this colour as the parameters of a true-colour SGR sequence,
// with base 38 for a foreground and 48 for a background.
func (c RGB) sgr(base int) string {
	return fmt.Sprintf("%d;2;%d;%d;%d", base, c.R, c.G, c.B)
}
