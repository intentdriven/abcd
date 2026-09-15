package scanner

import (
	"strings"
	"testing"
)

// TestNameContinuationIsAlphanumericOnly pins the ONE rule the home-path
// anchor uses for "the name goes on": a letter or digit continues it, and
// nothing else does. '.', '-' and '_' are boundaries — "/Users/me.zip",
// "/Users/me-old" and "/Users/me_snapshot" are the caller's name with a
// suffix, not another user, and main redacted every one of them; the alnum
// case ("/Users/alexandra" against "/Users/alex") is the only false positive
// the trailing anchor exists for.
func TestNameContinuationIsAlphanumericOnly(t *testing.T) {
	home := "/Users/zzhomeuser42" // abcd-audit:allow
	t.Setenv("HOME", home)
	sc, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	shapes := []string{
		"the archive is at https://ci.example.com" + home + ".zip for review",
		"the archive is at " + home + ".zip for review",
		"old copy under https://ci.example.com" + home + "-old/x here",
		"old copy under " + home + "-old/x here",
		"snapshot under /Volumes/T7" + home + "_snapshot/x here",
		"snapshot under " + home + "_snapshot/x here",
	}
	for _, text := range shapes {
		findings := sc.ScanText(text+"\n", "t")
		redacted, _ := Redact(text+"\n", findings)
		redacted = SweepCallerHome(redacted, home)
		redacted, resid := SurvivingCallerHome(redacted, home)
		if len(resid) > 0 {
			t.Errorf("%q: the write was refused instead of rewritten: %+v", text, resid)
		}
		if strings.Contains(redacted, "zzhomeuser42") {
			t.Errorf("%q: the caller's name reached the output:\n%s", text, redacted)
		}
		if !strings.Contains(redacted, "here") && !strings.Contains(redacted, "for review") {
			t.Errorf("%q: the rest of the line was lost:\n%s", text, redacted)
		}
	}
	// The alnum case is still another user's path, reported as such.
	text := "see /Users/zzhomeuser42andra/reports/q3.md\n" // abcd-audit:allow
	redacted, _ := Redact(text, sc.ScanText(text, "t"))
	if strings.Contains(redacted, "zzhomeuser42andra") || !strings.Contains(redacted, "[redacted-path]") {
		t.Errorf("the longer alnum name was not treated as another user's path:\n%s", redacted)
	}
}
