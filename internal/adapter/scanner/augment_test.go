package scanner

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// augValue is a value the native pattern set does not see (no prefix, no
// label, lowercase words), so a finding for it can only have come from the
// augmenter.
const augValue = "plumbob-harvest-quiet-lantern"

// fakeAug is a hand-rolled augmenter: it locates augValue on every line, or
// returns what the test scripted.
type fakeAug struct {
	avail    error
	scanErr  error // becomes Available()'s answer after a Scan
	findings func(text, file string) []Finding
	calls    int
}

func (f *fakeAug) Available() error { return f.avail }

func (f *fakeAug) Scan(text, file string) []Finding {
	f.calls++
	if f.scanErr != nil {
		f.avail = f.scanErr
		return nil
	}
	if f.findings != nil {
		return f.findings(text, file)
	}
	var out []Finding
	for i, ln := range strings.Split(text, "\n") {
		if c := strings.Index(ln, augValue); c >= 0 {
			out = append(out, Finding{File: file, Line: i + 1, Column: c + 1, Kind: "fake:value",
				Severity: SeverityInfo, Matched: augValue, Snippet: ln, Suggested: "echo " + augValue})
		}
	}
	return out
}

func newAug(t *testing.T, a Augmenter) *Scanner {
	t.Helper()
	sc, err := New(t.TempDir(), WithAugmenter(a))
	if err != nil {
		t.Fatal(err)
	}
	return sc
}

func TestScanTextAppendsTheAugmentersFindings(t *testing.T) {
	sc := newAug(t, &fakeAug{})
	text := "line one\nkey is " + augValue + " here\n"
	var got []Finding
	for _, f := range sc.ScanText(text, "doc.md") {
		if f.Matched == augValue {
			got = append(got, f)
		}
	}
	if len(got) != 1 {
		t.Fatalf("want one augmented finding, got %d: %+v", len(got), got)
	}
	f := got[0]
	if f.File != "doc.md" || f.Line != 2 || f.Column != 8 {
		t.Errorf("finding misplaced: %+v", f)
	}
	// The augmenter's finding is a secret whatever severity it claimed.
	if f.Severity != SeverityHardFail {
		t.Errorf("severity = %q, want hard_fail", f.Severity)
	}
	if f.Suggested != "" {
		t.Errorf("the augmenter's own suggestion text was kept: %q", f.Suggested)
	}
	// The native-only entry point does not run the augmenter.
	for _, n := range sc.ScanTextNative(text, "doc.md") {
		if n.Matched == augValue {
			t.Fatal("ScanTextNative reported an augmented finding")
		}
	}
	if got := UnsealedAugmented(text, sc.ScanText(text, "doc.md")); len(got) != 1 {
		t.Errorf("UnsealedAugmented on the raw text = %d findings, want 1", len(got))
	}
	red, _ := Redact(text, sc.ScanText(text, "doc.md"))
	if strings.Contains(red, augValue) {
		t.Fatalf("Redact left the augmented value: %q", red)
	}
	if got := UnsealedAugmented(red, sc.ScanText(text, "doc.md")); len(got) != 0 {
		t.Errorf("UnsealedAugmented on the redacted text = %+v, want none", got)
	}
}

func TestAugmentedFindingNeverEchoesTheValue(t *testing.T) {
	sc := newAug(t, &fakeAug{})
	text := "key is " + augValue + " here"
	for _, f := range sc.ScanText(text, "doc.md") {
		if f.Matched != augValue {
			continue
		}
		b, err := json.Marshal(f)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), augValue) {
			t.Fatalf("a serialised augmented finding echoes the value: %s", b)
		}
		return
	}
	t.Fatal("no augmented finding")
}

func TestAugmentedFindingsDeduplicateOnFileLineAndSpan(t *testing.T) {
	token := "AKIA" + strings.Repeat("Q", 16)
	text := "aws " + token
	// The augmenter reports the same span the native scanner already did,
	// twice, under its own kind.
	a := &fakeAug{findings: func(text, file string) []Finding {
		f := Finding{File: file, Line: 1, Column: 5, Kind: "gitleaks:aws", Matched: token}
		return []Finding{f, f}
	}}
	sc := newAug(t, a)
	n := 0
	for _, f := range sc.ScanText(text, "doc.md") {
		if f.Line == 1 && f.Column == 5 && f.Matched == token {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("the span is reported %d times, want once", n)
	}
}

func TestAugmentedFindingIsSanitised(t *testing.T) {
	text := "alpha " + augValue
	a := &fakeAug{findings: func(_, _ string) []Finding {
		return []Finding{{File: "../../elsewhere", Line: 1, Column: 7, Kind: "home_path_self\x1b[31m",
			Severity: SeverityInfo, Matched: augValue, Snippet: "raw " + augValue}}
	}}
	sc := newAug(t, a)
	var got *Finding
	for _, f := range sc.ScanText(text, "doc.md") {
		if f.Matched == augValue {
			got = &f
		}
	}
	if got == nil {
		t.Fatal("no augmented finding")
	}
	if got.File != "doc.md" {
		t.Errorf("the augmenter chose the file: %q", got.File)
	}
	if IsIdentityKind(got.Kind) || strings.ContainsAny(got.Kind, "\x1b[") || !strings.HasPrefix(got.Kind, "augmented:") {
		t.Errorf("kind not sanitised: %q", got.Kind)
	}
	if got.Snippet != snippet(text) {
		t.Errorf("snippet is the augmenter's, not the scanner's: %q", got.Snippet)
	}
}

func TestAugmentedFindingNotInTheTextDegradesTheScanner(t *testing.T) {
	for name, f := range map[string]Finding{
		"wrong bytes":  {Line: 1, Column: 1, Matched: "not-there"},
		"line too far": {Line: 9, Column: 1, Matched: "a"},
		"empty match":  {Line: 1, Column: 1},
		"column past":  {Line: 1, Column: 99, Matched: "a"},
	} {
		t.Run(name, func(t *testing.T) {
			f := f
			sc := newAug(t, &fakeAug{findings: func(_, _ string) []Finding { return []Finding{f} }})
			sc.ScanText("alpha", "doc.md")
			if bad, why := sc.Unavailable(); !bad || !strings.Contains(why, "augmenter") {
				t.Fatalf("Unavailable() = %v %q, want degraded naming the augmenter", bad, why)
			}
		})
	}
}

func TestAugmenterFindingCountIsBounded(t *testing.T) {
	a := &fakeAug{findings: func(_, file string) []Finding {
		out := make([]Finding, maxAugmentFindings+1)
		for i := range out {
			out[i] = Finding{File: file, Line: 1, Column: 1, Matched: "a"}
		}
		return out
	}}
	sc := newAug(t, a)
	sc.ScanText("alpha", "doc.md")
	if bad, _ := sc.Unavailable(); !bad {
		t.Fatal("an unbounded augmenter report did not degrade the scanner")
	}
}

func TestAugmenterScanFailureDegradesTheScanner(t *testing.T) {
	sc := newAug(t, &fakeAug{scanErr: errors.New("exit status 2\x1b]0;x\x07")})
	if bad, _ := sc.Unavailable(); bad {
		t.Fatal("degraded before any scan")
	}
	sc.ScanText("alpha", "doc.md")
	bad, why := sc.Unavailable()
	if !bad {
		t.Fatal("a failed augmenter run left the scanner trusted")
	}
	if strings.ContainsAny(why, "\x1b\x07") {
		t.Errorf("the augmenter's error reached the reason unsanitised: %q", why)
	}
	res, _ := sc.ScanBundle(nil)
	if !res.Unavailable {
		t.Error("ScanBundle does not report the degraded scanner")
	}
}

func TestAugmenterNotFoundIsAGapNotADegrade(t *testing.T) {
	a := &fakeAug{avail: fmt.Errorf("%w: fake not on PATH", ErrAugmenterNotFound)}
	sc := newAug(t, a)
	if bad, why := sc.Unavailable(); bad {
		t.Fatalf("not-found degraded the scanner: %s", why)
	}
	if gap := sc.AugmenterGap(); !strings.Contains(gap, "fake not on PATH") {
		t.Fatalf("AugmenterGap() = %q", gap)
	}
	sc.ScanText("alpha "+augValue, "doc.md")
	if a.calls != 0 {
		t.Error("a not-found augmenter was asked to scan")
	}
}

func TestAugmenterRefusedIsADegrade(t *testing.T) {
	sc := newAug(t, &fakeAug{avail: errors.New("configured path refused")})
	if bad, why := sc.Unavailable(); !bad || !strings.Contains(why, "configured path refused") {
		t.Fatalf("Unavailable() = %v %q", bad, why)
	}
	if sc.AugmenterGap() != "" {
		t.Error("a refused augmenter reads as a gap")
	}
}

func TestScanBundleAppendsAndFailsClosedOnTheGap(t *testing.T) {
	root := t.TempDir()
	p := writeFile(t, root, "doc.md", "key "+augValue+"\n")
	files := []BundleFile{{LogicalPath: "doc.md", ResolvedPath: p}}

	sc := newAug(t, &fakeAug{})
	res, _ := sc.ScanBundle(files)
	found := false
	for _, f := range res.Findings {
		found = found || f.Matched == augValue
	}
	if !found || res.HardFails < 1 {
		t.Fatalf("ScanBundle did not report the augmented finding: %+v", res)
	}

	gap := newAug(t, &fakeAug{avail: fmt.Errorf("%w: fake", ErrAugmenterNotFound)})
	res, _ = gap.ScanBundle(files)
	if res.HardFails != 1 || len(res.Unscanned) != 1 || !strings.Contains(res.UnscannedWhy[res.Unscanned[0]], "fake") {
		t.Fatalf("the gap did not fail the bundle closed: %+v", res)
	}
}

func TestDefaultAugmenterIsWiredThroughNew(t *testing.T) {
	a := &fakeAug{}
	restore := SetDefaultAugmenter(func(string) Augmenter { return a })
	defer restore()
	sc, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	sc.ScanText("alpha", "doc.md")
	if a.calls != 1 {
		t.Fatalf("the registered augmenter ran %d times, want 1", a.calls)
	}
	restore()
	sc, _ = New(t.TempDir())
	sc.ScanText("alpha", "doc.md")
	if a.calls != 1 {
		t.Fatal("the augmenter outlived its restore")
	}
}
