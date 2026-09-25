package surface

import (
	"strings"
	"testing"
)

// TestParseSentenceSplitsTheThreeClauses is itd-2609212113220149 criterion 1 on
// a good sentence: the doing clause runs to the colon, the writing clause to the
// semicolon, and the refusing clause to the closing period, in that order.
func TestParseSentenceSplitsTheThreeClauses(t *testing.T) {
	for _, tc := range []struct {
		in                    string
		does, writes, refuses string
	}{
		{
			in:      "List the issues in one status folder: Writes nothing; refuses when no status flag is given.",
			does:    "List the issues in one status folder",
			writes:  "Writes nothing",
			refuses: "refuses when no status flag is given",
		},
		{
			in:      "Check the machine's load before the tests start: Writes a run-log line in a run; never refuses.",
			does:    "Check the machine's load before the tests start",
			writes:  "Writes a run-log line in a run",
			refuses: "never refuses",
		},
	} {
		got, err := ParseSentence(tc.in)
		if err != nil {
			t.Fatalf("ParseSentence(%q) = %v, want the three clauses", tc.in, err)
		}
		if got.Does != tc.does || got.Writes != tc.writes || got.Refuses != tc.refuses {
			t.Errorf("ParseSentence(%q) = %+v, want %q / %q / %q", tc.in, got, tc.does, tc.writes, tc.refuses)
		}
	}
}

// TestParseSentenceNamesEachDefect is criterion 3's clause half: a sentence that
// lacks a clause, puts them out of order, runs past the cap or is not one
// sentence is refused, and the refusal names the defect rather than only
// failing.
func TestParseSentenceNamesEachDefect(t *testing.T) {
	long := "Render the " + strings.Repeat("very ", 40) + "long status: Writes nothing; refuses nothing it is given."
	for _, tc := range []struct {
		name, in, want string
	}{
		{"empty", "", "empty"},
		{"no writing clause", "List the issues; refuses when no status flag is given.", "colon"},
		{"no refusing clause", "List the issues: Writes nothing.", "semicolon"},
		{"no doing clause", ": Writes nothing; refuses always.", "doing clause"},
		{"writing clause not opened", "List the issues: Nothing is written; refuses when no flag is given.", `"Writes"`},
		{"refusing clause not opened", "List the issues: Writes nothing; exits 2 without a flag.", `"refuses"`},
		{"clauses out of order", "List the issues; refuses without a flag: Writes nothing.", "colon"},
		{"a second colon", "List the issues: Writes nothing: ever; refuses without a flag.", "one colon"},
		{"a second semicolon", "List the issues: Writes nothing; refuses without a flag; exits 2.", "one semicolon"},
		{"no closing period", "List the issues: Writes nothing; refuses without a flag", "period"},
		{"two sentences", "List the issues. Then stop: Writes nothing; refuses without a flag.", "one sentence"},
		{"lower-case opening", "list the issues: Writes nothing; refuses without a flag.", "capital"},
		{"a line break", "List the issues:\nWrites nothing; refuses without a flag.", "line"},
		{"over the cap", long, "160"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseSentence(tc.in)
			if err == nil {
				t.Fatalf("ParseSentence(%q) = nil, want a refusal naming %q", tc.in, tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ParseSentence(%q) = %q, want it to name %q", tc.in, err, tc.want)
			}
		})
	}
}

// TestSentenceCapCountsCharactersNotBytes: the cap is declared in characters,
// so a sentence carrying a multi-byte character is measured the way a reader
// counts it, and one exactly at the cap passes.
func TestSentenceCapCountsCharactersNotBytes(t *testing.T) {
	head := "Show the section×repo table: Writes nothing; refuses "
	tail := "."
	pad := SentenceCap - len([]rune(head)) - len([]rune(tail))
	at := head + strings.Repeat("x", pad) + tail
	if n := len([]rune(at)); n != SentenceCap {
		t.Fatalf("fixture is %d characters, want %d", n, SentenceCap)
	}
	if _, err := ParseSentence(at); err != nil {
		t.Fatalf("a sentence of exactly %d characters = %v, want it accepted", SentenceCap, err)
	}
	if _, err := ParseSentence(head + strings.Repeat("x", pad+1) + tail); err == nil {
		t.Fatalf("a sentence one character over the cap was accepted")
	}
}

// TestEverySentenceInTheManifestParses is the manifest's own form check: every
// sentence the table declares parses into its three clauses under the cap, so a
// badly formed entry is named here, before any front door renders it.
func TestEverySentenceInTheManifestParses(t *testing.T) {
	paths := SentencePaths()
	if len(paths) == 0 {
		t.Fatal("the sentence manifest is empty; the check would pass vacuously")
	}
	for _, p := range paths {
		s, _ := SentenceFor(p)
		if _, err := ParseSentence(s); err != nil {
			t.Errorf("%s: %v", p, err)
		}
	}
}

// TestEncodeRecordsTheSentence: the snapshot carries each command's sentence,
// so the committed surface holds the one line its three renders come from, and
// a command with none records no key.
func TestEncodeRecordsTheSentence(t *testing.T) {
	snap := NewSnapshot([]Command{
		{Path: "abcd capture", Sentence: "Capture an issue: Writes a record; refuses outside a checkout."},
		{Path: "abcd hook", Hidden: true},
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
      "sentence": "Capture an issue: Writes a record; refuses outside a checkout.",
      "flags": []
    },
    {
      "path": "abcd hook",
      "hidden": true,
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
		t.Fatalf("Decode(version 3) = %v", err)
	}
	if back.Commands[0].Sentence != snap.Commands[0].Sentence {
		t.Fatalf("round trip lost the sentence: %+v", back.Commands[0])
	}
}

// TestDecodeReadsTheVersionTwoShape keeps the last released shape readable: the
// release guardrail reads its baseline out of the last tag, which carries a
// version-2 file with no sentence recorded.
func TestDecodeReadsTheVersionTwoShape(t *testing.T) {
	v2 := `{"schema_version":2,"commands":[{"path":"abcd capture","hidden":false,` +
		`"group":"records","block":"people","flags":[]}],"manifest":[]}`
	snap, err := Decode([]byte(v2))
	if err != nil {
		t.Fatalf("Decode(version 2) = %v, want the released shape to stay readable", err)
	}
	if snap.Commands[0].Sentence != "" || snap.Commands[0].Group != "records" {
		t.Fatalf("Decode(version 2) = %+v", snap.Commands[0])
	}
}
