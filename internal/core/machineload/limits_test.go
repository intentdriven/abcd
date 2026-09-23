package machineload

import (
	"strings"
	"testing"
)

// TestDefaultLimitsFromCores: 30 minutes, and four times the online core count.
func TestDefaultLimitsFromCores(t *testing.T) {
	for cores, want := range map[int]float64{16: 64, 10: 40, 1: 4} {
		l := DefaultLimits(cores)
		if l.StrayMinutes != 30 || l.ExtremeLoad != want || l.StrayFromFile || l.ExtremeFromFile {
			t.Errorf("DefaultLimits(%d) = %+v, want stray 30, extreme %v", cores, l, want)
		}
	}
}

// TestLimitsFileSetsEitherOrBoth: each key sets its own limit, an omitted key
// takes its default, and comments and blank lines are ignored.
func TestLimitsFileSetsEitherOrBoth(t *testing.T) {
	cases := []struct {
		name, file       string
		stray            int
		extreme          float64
		straySet, extSet bool
	}{
		{"stray only", "stray-minutes 45\n", 45, 64, true, false},
		{"extreme only", "# mine\n\nextreme-load\t80.5\n", 30, 80.5, false, true},
		{"both", "# abcd's load check\nstray-minutes 10\nextreme-load 100\n", 10, 100, true, true},
		{"empty", "# nothing set\n", 30, 64, false, false},
		{"bounds", "stray-minutes 10080\nextreme-load 100000\n", 10080, 100000, true, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			l, fault := ParseLimits([]byte(c.file), 16)
			if fault != nil {
				t.Fatalf("fault: %v", fault)
			}
			if l.StrayMinutes != c.stray || l.ExtremeLoad != c.extreme || l.StrayFromFile != c.straySet || l.ExtremeFromFile != c.extSet {
				t.Fatalf("limits = %+v", l)
			}
		})
	}
}

// TestMalformedLimitsFallBackWhole: any fault makes the whole file unusable, so
// both limits take their defaults, never a mix, and the report names the line
// and the fault class but never the value.
func TestMalformedLimitsFallBackWhole(t *testing.T) {
	const secret = "98765"
	cases := []struct {
		name, file, class string
		line              int
	}{
		{"unknown key", "extreme-load 50\nstray-mins 12\n", `unknown key "stray-mins" (known: stray-minutes, extreme-load)`, 2},
		{"repeated key", "stray-minutes 12\nstray-minutes 13\n", "stray-minutes is given twice", 2},
		{"non-numeric", "stray-minutes 12\nextreme-load lots" + secret + "\n", "the value of extreme-load is not a number", 2},
		{"fraction for minutes", "stray-minutes 1.5\n", "the value of stray-minutes is not a whole number", 1},
		{"zero", "extreme-load 50\n\nstray-minutes 0\n", "stray-minutes is out of range (1 to 10080)", 3},
		{"out of range", "extreme-load " + secret + "0\n", "extreme-load is out of range (above 0, at most 100000)", 1},
		{"no value", "extreme-load 50\nstray-minutes\n", "stray-minutes has no value", 2},
		{"trailing field", "stray-minutes 12 " + secret + "\n", "stray-minutes has more than one value", 1},
		{"not a number at all", "extreme-load NaN\n", "the value of extreme-load is not a number", 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			l, fault := ParseLimits([]byte(c.file), 16)
			if fault == nil {
				t.Fatalf("no fault for %q", c.file)
			}
			if l != DefaultLimits(16) {
				t.Fatalf("limits = %+v, want both defaults", l)
			}
			if fault.Line != c.line || fault.Class != c.class {
				t.Fatalf("fault = line %d %q, want line %d %q", fault.Line, fault.Class, c.line, c.class)
			}
			if strings.Contains(fault.Error(), secret) {
				t.Fatalf("the fault echoes a value: %q", fault.Error())
			}
		})
	}

	// An offending key is echoed made safe and capped at 32 bytes.
	_, fault := ParseLimits([]byte("\x1b[31m"+strings.Repeat("k", 60)+" 1\n"), 16)
	if fault == nil || strings.ContainsRune(fault.Class, 0x1b) || strings.Contains(fault.Class, strings.Repeat("k", 33)) {
		t.Fatalf("hostile key echoed raw: %+v", fault)
	}
}
