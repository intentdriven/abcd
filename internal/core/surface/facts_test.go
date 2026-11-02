package surface

import (
	"reflect"
	"testing"
)

func TestFlagKindFoldsTheLibraryTypes(t *testing.T) {
	for lib, want := range map[string]string{
		"bool": FlagBool, "string": FlagString, "int": FlagInt, "int64": FlagInt, "uint": FlagInt, "count": FlagInt,
		"float64": FlagNumber, "duration": FlagDuration,
		"stringSlice": FlagList, "stringArray": FlagList, "intSlice": FlagList,
		"ipNet": "ipNet",
	} {
		if got := FlagKind(lib); got != want {
			t.Errorf("FlagKind(%q) = %q, want %q", lib, got, want)
		}
	}
}

func TestZeroDefaultIsTheKindsOwnZero(t *testing.T) {
	cases := []struct {
		kind, def string
		zero      bool
	}{
		{FlagString, "", true},
		{FlagBool, "false", true},
		{FlagBool, "true", false},
		{FlagInt, "0", true},
		{FlagInt, "8", false},
		{FlagList, "[]", true},
		{FlagList, "[a]", false},
		{FlagDuration, "0s", true},
		{FlagDuration, "5m0s", false},
		// A string flag defaulting to the text "false" or "0" keeps it.
		{FlagString, "false", false},
		{FlagString, "0", false},
	}
	for _, c := range cases {
		if got := ZeroDefault(c.kind, c.def); got != c.zero {
			t.Errorf("ZeroDefault(%q, %q) = %v, want %v", c.kind, c.def, got, c.zero)
		}
	}
}

func TestCanonicalFactsSortsWithoutTouchingItsInput(t *testing.T) {
	in := []Facts{
		{Path: "abcd z", Flags: []FlagFact{{Name: "b"}, {Name: "a", Allowed: []string{"y", "x"}}},
			Inputs: []InputFact{{Flag: "z-json"}, {Flag: "a-json"}}, Subcommands: []string{"q", "p"}},
		{Path: "abcd a"},
	}
	out := CanonicalFacts(in)
	if out[0].Path != "abcd a" || out[1].Path != "abcd z" {
		t.Fatalf("paths not sorted: %s, %s", out[0].Path, out[1].Path)
	}
	if out[1].Flags[0].Name != "a" || out[1].Inputs[0].Flag != "a-json" || out[1].Subcommands[0] != "p" {
		t.Errorf("a command's collections are not sorted: %+v", out[1])
	}
	// A closed set keeps its declared order: it is the reader's slice, not a
	// collection of the walk's.
	if !reflect.DeepEqual(out[1].Flags[0].Allowed, []string{"y", "x"}) {
		t.Errorf("the closed set was reordered: %v", out[1].Flags[0].Allowed)
	}
	if out[0].Flags == nil || out[0].Inputs == nil {
		t.Error("a command with no flags or inputs encodes them as null, not as an empty list")
	}
	if in[0].Path != "abcd z" || in[0].Flags[0].Name != "b" || in[0].Subcommands[0] != "q" {
		t.Errorf("CanonicalFacts sorted its input in place: %+v", in[0])
	}
	out[1].Flags[0].Allowed[0] = "changed"
	if in[0].Flags[1].Allowed[0] != "y" {
		t.Error("the canonical facts share a closed set's backing array with the input")
	}
}
