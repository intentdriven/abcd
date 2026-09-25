package surface

import (
	"strings"
	"testing"
)

// TestNewSnapshotSortsEveryCollection is the determinism detector for the
// constructor: callers hand it whatever order their walk produced (cobra's
// registration order, a map range), and the snapshot must come back in one
// canonical order regardless.
func TestNewSnapshotSortsEveryCollection(t *testing.T) {
	snap := NewSnapshot(
		[]Command{
			{Path: "abcd intent", Flags: []Flag{{Name: "json"}, {Name: "all"}}},
			{Path: "abcd", Flags: nil},
			{Path: "abcd hook", Hidden: true},
		},
		[]ManifestEntry{
			{File: "b.json", Key: "name"},
			{File: "a.json", Key: "zeta"},
			{File: "a.json", Key: "alpha"},
		},
	)

	if snap.SchemaVersion != SchemaVersion {
		t.Fatalf("schema version = %d, want %d", snap.SchemaVersion, SchemaVersion)
	}
	wantPaths := []string{"abcd", "abcd hook", "abcd intent"}
	if got := commandPaths(snap); !equalStrings(got, wantPaths) {
		t.Fatalf("command order = %v, want %v", got, wantPaths)
	}
	wantFlags := []string{"all", "json"}
	var gotFlags []string
	for _, f := range snap.Commands[2].Flags {
		gotFlags = append(gotFlags, f.Name)
	}
	if !equalStrings(gotFlags, wantFlags) {
		t.Fatalf("flag order = %v, want %v", gotFlags, wantFlags)
	}
	wantEntries := []string{"a.json:alpha", "a.json:zeta", "b.json:name"}
	var gotEntries []string
	for _, e := range snap.Manifest {
		gotEntries = append(gotEntries, e.File+":"+e.Key)
	}
	if !equalStrings(gotEntries, wantEntries) {
		t.Fatalf("manifest order = %v, want %v", gotEntries, wantEntries)
	}
}

// TestNewSnapshotCopiesInput proves the constructor does not alias the caller's
// slices: a snapshot that a later mutation can reorder is not a snapshot.
func TestNewSnapshotCopiesInput(t *testing.T) {
	cmds := []Command{{Path: "abcd b"}, {Path: "abcd a"}}
	entries := []ManifestEntry{{File: "b.json"}, {File: "a.json"}}
	snap := NewSnapshot(cmds, entries)

	cmds[0].Path = "mutated"
	entries[0].File = "mutated"

	if snap.Commands[1].Path != "abcd b" {
		t.Fatalf("commands aliased the caller's slice: %v", commandPaths(snap))
	}
	if snap.Manifest[1].File != "b.json" {
		t.Fatalf("manifest aliased the caller's slice: %+v", snap.Manifest)
	}
}

// TestEncodeShape pins the JSON contract the guardrail diffs: fixed key order,
// two-space indent, empty collections as [] rather than null, and exactly one
// trailing newline. Downstream stages read this shape, so a change here is a
// change to a published artefact.
func TestEncodeShape(t *testing.T) {
	snap := NewSnapshot(
		[]Command{
			{Path: "abcd", Flags: []Flag{{Name: "json", Type: "bool"}}},
			{Path: "abcd hook", Hidden: true},
		},
		[]ManifestEntry{{File: ".claude-plugin/plugin.json", Key: "name"}},
	)

	got, err := Encode(snap)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	want := `{
  "schema_version": 2,
  "commands": [
    {
      "path": "abcd",
      "hidden": false,
      "flags": [
        {
          "name": "json",
          "shorthand": "",
          "type": "bool",
          "required": false,
          "hidden": false
        }
      ]
    },
    {
      "path": "abcd hook",
      "hidden": true,
      "flags": []
    }
  ],
  "manifest": [
    {
      "file": ".claude-plugin/plugin.json",
      "key": "name"
    }
  ]
}
`
	if string(got) != want {
		t.Fatalf("Encode shape mismatch:\ngot:\n%s\nwant:\n%s", got, want)
	}
	if !strings.HasSuffix(string(got), "}\n") || strings.HasSuffix(string(got), "}\n\n") {
		t.Fatalf("Encode must end in exactly one trailing newline, got %q", tail(string(got)))
	}
}

// TestEncodeCanonicalisesUnsortedInput is the second determinism detector: Encode
// is the boundary that writes the committed artefact, so it must not depend on
// the caller having gone through NewSnapshot.
func TestEncodeCanonicalisesUnsortedInput(t *testing.T) {
	unsorted := Snapshot{
		SchemaVersion: SchemaVersion,
		Commands:      []Command{{Path: "abcd zeta"}, {Path: "abcd alpha"}},
		Manifest:      []ManifestEntry{{File: "b.json", Key: "k"}, {File: "a.json", Key: "k"}},
	}
	sorted := NewSnapshot(unsorted.Commands, unsorted.Manifest)

	fromUnsorted, err := Encode(unsorted)
	if err != nil {
		t.Fatalf("Encode(unsorted): %v", err)
	}
	fromSorted, err := Encode(sorted)
	if err != nil {
		t.Fatalf("Encode(sorted): %v", err)
	}
	if string(fromUnsorted) != string(fromSorted) {
		t.Fatalf("Encode is order-sensitive:\nunsorted:\n%s\nsorted:\n%s", fromUnsorted, fromSorted)
	}
}

// TestEncodeIsRepeatable runs the encoder many times over one snapshot: any
// map-ranging or other non-determinism inside the encoder shows up as a diff.
func TestEncodeIsRepeatable(t *testing.T) {
	snap := NewSnapshot(
		[]Command{{Path: "abcd", Flags: []Flag{{Name: "json"}, {Name: "all"}, {Name: "dry-run"}}}},
		[]ManifestEntry{{File: "a.json", Key: "x"}, {File: "a.json", Key: "y"}},
	)
	first, err := Encode(snap)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	for i := 0; i < 50; i++ {
		again, err := Encode(snap)
		if err != nil {
			t.Fatalf("Encode iteration %d: %v", i, err)
		}
		if string(again) != string(first) {
			t.Fatalf("Encode is not deterministic at iteration %d", i)
		}
	}
}

// TestDecodeRoundTrip proves the committed artefact reads back as the value the
// guardrail diffs — without a decoder the JSON shape would be write-only.
func TestDecodeRoundTrip(t *testing.T) {
	snap := NewSnapshot(
		[]Command{{Path: "abcd hook", Hidden: true, Flags: []Flag{{Name: "since", Shorthand: "s", Type: "string", Required: true}}}},
		[]ManifestEntry{{File: ".claude-plugin/plugin.json", Key: "author.name"}},
	)
	data, err := Encode(snap)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	back, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	again, err := Encode(back)
	if err != nil {
		t.Fatalf("Encode(back): %v", err)
	}
	if string(again) != string(data) {
		t.Fatalf("round trip lost data:\ngot:\n%s\nwant:\n%s", again, data)
	}
}

// TestDecodeRefusesBadInput keeps the guardrail fail-closed: a baseline it cannot
// read faithfully must be an error, never a silently empty surface that would
// make every removal invisible.
func TestDecodeRefusesBadInput(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"malformed", `{`, "decoding"},
		{"unknown field", `{"schema_version":1,"commands":[],"manifest":[],"extra":1}`, "extra"},
		{"future schema", `{"schema_version":99,"commands":[],"manifest":[]}`, "schema version"},
		{"missing schema", `{"commands":[],"manifest":[]}`, "schema version"},
		{
			// A stream decoder stops at the first value, so anything after the
			// snapshot would be accepted silently. The artefact is exactly one
			// JSON document; bytes beyond it mean the blob is not the artefact.
			"trailing content",
			`{"schema_version":1,"commands":[],"manifest":[]}` + "\n{\"schema_version\":1}\n",
			"trailing",
		},
		{
			"trailing garbage",
			`{"schema_version":1,"commands":[],"manifest":[]}` + "\n<<<not json at all>>>\n",
			"trailing",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Decode([]byte(tc.in))
			if err == nil {
				t.Fatalf("Decode(%q) = nil error, want one", tc.in)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Decode(%q) error = %q, want it to mention %q", tc.in, err, tc.want)
			}
		})
	}
}

func commandPaths(s Snapshot) []string {
	var out []string
	for _, c := range s.Commands {
		out = append(out, c.Path)
	}
	return out
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func tail(s string) string {
	if len(s) < 8 {
		return s
	}
	return s[len(s)-8:]
}
