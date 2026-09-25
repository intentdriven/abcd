package surface

import (
	"strings"
	"testing"
)

// moved_test.go holds itd-2609212130136102 criterion 4 on the data side: every
// moved spelling is recorded in the snapshot with its successor, and the
// generated appendix names the new forms only.

// TestEncodeRecordsMovedTo: a moved spelling carries its successor under
// `moved_to`, a command that did not move records no key, and the shape round
// trips at the version that introduced the field.
func TestEncodeRecordsMovedTo(t *testing.T) {
	snap := NewSnapshot([]Command{
		{Path: "abcd docs lint", MovedTo: "abcd lint docs"},
		{Path: "abcd lint docs"},
	}, nil)
	got, err := Encode(snap)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	want := `{
  "schema_version": 4,
  "commands": [
    {
      "path": "abcd docs lint",
      "hidden": false,
      "moved_to": "abcd lint docs",
      "flags": []
    },
    {
      "path": "abcd lint docs",
      "hidden": false,
      "flags": []
    }
  ],
  "manifest": []
}
`
	if string(got) != want {
		t.Fatalf("Encode shape mismatch:\ngot:\n%s\nwant:\n%s", got, want)
	}
	back, err := Decode(got)
	if err != nil {
		t.Fatalf("Decode(version 4) = %v", err)
	}
	if back.Commands[0].MovedTo != "abcd lint docs" {
		t.Fatalf("round trip lost moved_to: %+v", back.Commands[0])
	}
}

// TestDecodeReadsTheVersionThreeShape keeps the last released shape readable: the
// release guardrail reads its baseline out of the last tag, which carries a
// version-3 file with no successor recorded.
func TestDecodeReadsTheVersionThreeShape(t *testing.T) {
	v3 := `{"schema_version":3,"commands":[{"path":"abcd version","hidden":false,` +
		`"sentence":"Print it: Writes nothing; refuses any argument.","flags":[]}],"manifest":[]}`
	snap, err := Decode([]byte(v3))
	if err != nil {
		t.Fatalf("Decode(version 3) = %v, want the released shape to stay readable", err)
	}
	if snap.Commands[0].MovedTo != "" {
		t.Fatalf("Decode(version 3) = %+v", snap.Commands[0])
	}
}

// movedTree is a tree holding both kinds of move: leaves that moved whole
// (`ahoy dry-run`, `version`) and a parent whose bare form moved while its
// sub-verb stays (`ahoy remote`, whose `apply` is live).
func movedTree() []Command {
	return []Command{
		{Path: "abcd", Flags: []Flag{{Name: "version", Type: "bool"}}},
		{Path: "abcd ahoy", Flags: []Flag{{Name: "dry-run", Type: "bool"}, {Name: "remote", Type: "bool"}}},
		{Path: "abcd ahoy doctor"},
		{Path: "abcd ahoy dry-run", MovedTo: "abcd ahoy --dry-run"},
		{Path: "abcd ahoy remote", MovedTo: "abcd ahoy --remote"},
		{Path: "abcd ahoy remote apply", Flags: []Flag{{Name: "yes", Type: "bool"}}},
		{Path: "abcd version", MovedTo: "abcd --version", Flags: []Flag{{Name: "check", Type: "bool"}}},
	}
}

// TestComposeAppendixNamesTheNewFormsOnly: a leaf that moved whole is listed
// nowhere — neither as a section nor among its parent's sub-verbs — and a
// parent whose bare form moved keeps its section, with its live sub-verbs and a
// line naming where the bare form went.
func TestComposeAppendixNamesTheNewFormsOnly(t *testing.T) {
	got := ComposeAppendix([]string{"abcd ahoy"}, movedTree())
	for _, want := range []string{
		"Sub-verbs: `abcd ahoy doctor`, `abcd ahoy remote`.",
		"### `abcd ahoy remote`",
		"Bare, it moved to `abcd ahoy --remote`.",
		"### `abcd ahoy remote apply`",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("appendix lacks %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "ahoy dry-run") {
		t.Errorf("appendix names the moved leaf `abcd ahoy dry-run`:\n%s", got)
	}
}

// TestComposeAppendixForAMovedVerbNamesItsSuccessor: a chapter whose own
// command moved whole says where it went and lists none of the stub's flags.
func TestComposeAppendixForAMovedVerbNamesItsSuccessor(t *testing.T) {
	got := ComposeAppendix([]string{"abcd version"}, movedTree())
	if !strings.Contains(got, "### `abcd version`") || !strings.Contains(got, "It moved to `abcd --version`.") {
		t.Fatalf("appendix does not name the successor:\n%s", got)
	}
	if strings.Contains(got, "--check") || strings.Contains(got, "| Flag |") {
		t.Fatalf("appendix lists the stub's flags:\n%s", got)
	}
}
