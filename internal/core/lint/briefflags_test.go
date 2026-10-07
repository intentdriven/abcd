package lint

import (
	"errors"
	"strings"
	"testing"
)

func briefFlagConfig() Config {
	return Config{Rules: map[string]RuleConfig{ruleBriefFlagLanded: {Enabled: true, Severity: "blocker"}}}
}

// brief_flag_landed reports each flag the registered reader names, on the
// flag's line, saying what the flag must record (iss-2610050259233425).
func TestBriefFlagLandedReportsEachMiss(t *testing.T) {
	t.Cleanup(func() { SetBriefFlagCheck(nil) })
	SetBriefFlagCheck(func(string) ([]BriefFlagMiss, error) {
		return []BriefFlagMiss{{File: briefFlagsFile, Line: 7, Chapter: "10-docs.md",
			Replacement: "the drafted wording", Why: "no line of the chapter contains"}}, nil
	})
	got, err := Lint(briefFlagConfig(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %+v, want one finding", got)
	}
	f := got[0]
	if f.RuleID != ruleBriefFlagLanded || f.Severity != "blocker" || f.File != briefFlagsFile || f.Line != 7 ||
		!strings.Contains(f.Message, "10-docs.md") || !strings.Contains(f.Message, `"the drafted wording"`) ||
		!strings.Contains(f.Message, "sentence as committed") {
		t.Fatalf("finding %+v", f)
	}
}

// Unregistered, the rule says so rather than reporting nothing; a reader that
// cannot read the flags fails the lint rather than passing it.
func TestBriefFlagLandedUnregisteredAndUnreadable(t *testing.T) {
	t.Cleanup(func() { SetBriefFlagCheck(nil) })
	SetBriefFlagCheck(nil)
	got, err := Lint(briefFlagConfig(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !strings.Contains(got[0].Message, "SetBriefFlagCheck") {
		t.Fatalf("unregistered reader: got %+v, want one finding naming lint.SetBriefFlagCheck", got)
	}
	SetBriefFlagCheck(func(string) ([]BriefFlagMiss, error) { return nil, errors.New("schema_version 2, want 1") })
	if _, err := Lint(briefFlagConfig(), t.TempDir()); err == nil || !strings.Contains(err.Error(), "schema_version 2") {
		t.Fatalf("an unreadable flags file did not fail the lint: %v", err)
	}
}
