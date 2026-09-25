package surface

import (
	"strings"
	"testing"
)

// TestEncodeRecordsGroupAndBlock pins the two placement fields itd-146 adds to
// the committed command tree: a visible top-level verb carries the help group it
// is listed under and the block that group belongs to, a sub-verb listed in the
// agents block carries the block alone, and every other command carries neither
// key, so the artefact grows by exactly the placements and nothing else.
func TestEncodeRecordsGroupAndBlock(t *testing.T) {
	snap := NewSnapshot([]Command{
		{Path: "abcd capture", Group: "records", Block: "people"},
		{Path: "abcd capture list"},
		{Path: "abcd guard hook", Block: "agents"},
	}, nil)
	got, err := Encode(snap)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	want := `{
  "schema_version": 3,
  "commands": [
    {
      "path": "abcd capture",
      "hidden": false,
      "group": "records",
      "block": "people",
      "flags": []
    },
    {
      "path": "abcd capture list",
      "hidden": false,
      "flags": []
    },
    {
      "path": "abcd guard hook",
      "hidden": false,
      "block": "agents",
      "flags": []
    }
  ],
  "manifest": []
}
`
	if string(got) != want {
		t.Fatalf("Encode shape mismatch:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

// TestDecodeReadsTheReleasedVersionOneShape is the compatibility half of the
// schema bump. The release guardrail reads its baseline out of the LAST RELEASE
// TAG, and every tag cut before itd-146 carries a version-1 snapshot. A decoder
// that knew only version 2 would turn the first cut after this change into a
// decode error, so version 1 stays readable: it is version 2 with no placement
// recorded, which is exactly what those releases shipped.
func TestDecodeReadsTheReleasedVersionOneShape(t *testing.T) {
	v1 := `{"schema_version":1,"commands":[{"path":"abcd","hidden":false,"flags":[]},` +
		`{"path":"abcd capture","hidden":false,"flags":[]}],"manifest":[]}`
	snap, err := Decode([]byte(v1))
	if err != nil {
		t.Fatalf("Decode(version 1) = %v, want the released shape to stay readable", err)
	}
	if len(snap.Commands) != 2 || snap.Commands[1].Group != "" || snap.Commands[1].Block != "" {
		t.Fatalf("Decode(version 1) = %+v, want two commands with no placement", snap.Commands)
	}
}

// TestPlacementChangesNamesEachMovedVerb is the detector behind itd-146's fourth
// criterion: a verb whose group or block changed must be NAMED, because the
// release gate's byte comparison alone says only that the snapshot differs
// somewhere in a file of several thousand lines. Commands present on one side
// only are additions or removals, which the break taxonomy and the drift test
// already report; they are not placement changes and are not listed here.
func TestPlacementChangesNamesEachMovedVerb(t *testing.T) {
	committed := NewSnapshot([]Command{
		{Path: "abcd capture", Group: "records", Block: "people"},
		{Path: "abcd guard hook", Block: "agents"},
		{Path: "abcd lint", Group: "checks", Block: "people"},
		{Path: "abcd gone", Group: "records", Block: "people"},
	}, nil)
	current := NewSnapshot([]Command{
		{Path: "abcd capture", Group: "checks", Block: "people"},
		{Path: "abcd guard hook", Block: "people"},
		{Path: "abcd lint", Group: "checks", Block: "people"},
		{Path: "abcd new", Group: "records", Block: "people"},
	}, nil)

	got := PlacementChanges(committed, current)
	want := []string{
		"abcd capture: group records → checks",
		"abcd guard hook: block agents → people",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("PlacementChanges =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if again := PlacementChanges(committed, committed); len(again) != 0 {
		t.Fatalf("PlacementChanges(same, same) = %v, want none", again)
	}
}
