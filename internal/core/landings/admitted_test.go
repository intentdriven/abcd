package landings

import (
	"testing"
)

const ruleRel = ".abcd/development/decisions/adrs/2609291342092738-drain-eligibility.md"

// TestTheTrailerRoundTrips: what the loop composes is what the reader reads
// back, for each kind of lane, beside the trailers a records commit carries.
func TestTheTrailerRoundTrips(t *testing.T) {
	for _, a := range []Admitted{
		{Kind: KindDrain, Record: "adr-2609291342092738", Path: ruleRel},
		{Kind: KindBuild, Record: "spc-2610301950480609"},
		{Kind: KindBuild, Record: "iss-2611011230226453"},
	} {
		if why := a.Check(); why != "" {
			t.Fatalf("%+v is an admission the loop writes: %s", a, why)
		}
		msg := "chore(record): land lane-1 of run-1\n\nThe implement loop's landing.\n\n" +
			"Resolves: iss-1\n" + a.Trailer() + "\nAssisted-by: abcd:dev\n"
		got, ok := ParseAdmitted(msg)
		want := a
		want.Path = ""
		if !ok || got != want {
			t.Fatalf("ParseAdmitted(%q) = %+v %v, want %+v", msg, got, ok, want)
		}
	}
	if got := (Admitted{Kind: KindDrain, Record: "adr-7", Path: ".abcd/development/decisions/adrs/0007-x.md"}).Trailer(); got != "Admitted-by: drain adr-7" {
		t.Fatalf("the trailer is %q", got)
	}
}

// TestCheckRefusesWhatTheLoopNeverWrites: an admission is held to the shapes
// the loop writes, since its value reaches a commit message.
func TestCheckRefusesWhatTheLoopNeverWrites(t *testing.T) {
	for name, a := range map[string]Admitted{
		"no kind":                   {Record: "spc-1"},
		"an unknown kind":           {Kind: "merge", Record: "spc-1"},
		"a drain with no record":    {Kind: KindDrain, Path: ruleRel},
		"the bundled baseline":      {Kind: KindDrain},
		"a drain with no path":      {Kind: KindDrain, Record: "adr-2609291342092738"},
		"a path outside the store":  {Kind: KindDrain, Record: "adr-2609291342092738", Path: "docs/2609291342092738-x.md"},
		"a path to another record":  {Kind: KindDrain, Record: "adr-2609291342092738", Path: ".abcd/development/decisions/adrs/0007-x.md"},
		"a padded ADR id":           {Kind: KindDrain, Record: "adr-0007", Path: ".abcd/development/decisions/adrs/0007-x.md"},
		"a build of an intent":      {Kind: KindBuild, Record: "itd-1"},
		"a build with a line break": {Kind: KindBuild, Record: "spc-1\nResolves: iss-2"},
		"a build with a path":       {Kind: KindBuild, Record: "spc-1", Path: ruleRel},
	} {
		if a.Check() == "" {
			t.Errorf("%s: %+v is refused", name, a)
		}
	}
}

// TestParseAdmittedReadsOnlyTheTrailerBlock: a line in the prose is not a
// trailer, and a block carrying two admissions or a malformed one carries none.
func TestParseAdmittedReadsOnlyTheTrailerBlock(t *testing.T) {
	for name, msg := range map[string]string{
		"none":         "fix: a thing\n\nAssisted-by: None\n",
		"in the prose": "fix: a thing\n\nAdmitted-by: build spc-1\n\nAssisted-by: None\n",
		"two differ":   "fix: a thing\n\nAdmitted-by: build spc-1\nAdmitted-by: build spc-2\n",
		"malformed":    "fix: a thing\n\nAdmitted-by: build itd-1\n",
		"no record":    "fix: a thing\n\nAdmitted-by: drain\n",
	} {
		if a, ok := ParseAdmitted(msg); ok {
			t.Errorf("%s: %q carries no admission, read %+v", name, msg, a)
		}
	}
	if a, ok := ParseAdmitted("x\n\nadmitted-by: build spc-1\nAdmitted-by: build spc-1\n"); !ok || a.Record != "spc-1" {
		t.Errorf("a repeated admission, its key in any case, is read: %+v %v", a, ok)
	}
}
