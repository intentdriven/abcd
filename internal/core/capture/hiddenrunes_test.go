package capture

import (
	"errors"
	"strings"
	"testing"
)

// hiddenRunes is one of each class the record-write boundary must encode: a
// bidi override (Trojan-Source reordering), a zero-width space, a C1 control
// and DEL. Each reached a committed record verbatim (iss-2608301206073609).
var hiddenRunes = []string{"‮", "​", "\u0085", "\x7f"}

// assertNoHiddenRune fails when any hidden rune survives into the record, and
// when its percent-encoded form is absent — encoded, not dropped, so the record
// stays lossless.
func assertNoHiddenRune(t *testing.T, what, raw string) {
	t.Helper()
	for _, r := range hiddenRunes {
		if strings.Contains(raw, r) {
			t.Errorf("%s: the hidden rune %q reached the committed record verbatim:\n%q", what, r, raw)
		}
	}
	for _, enc := range []string{"%E2%80%AE", "%E2%80%8B", "%C2%85", "%7F"} {
		if !strings.Contains(raw, enc) {
			t.Errorf("%s: the encoded form %s is missing, so a rune was dropped rather than encoded:\n%q", what, enc, raw)
		}
	}
}

func hiddenText(prefix string) string {
	return prefix + " one‮two three​four five\u0085six seven\x7feight"
}

func TestCaptureBodyEncodesHiddenRunes(t *testing.T) {
	repo, ir := ledger(t)
	res, err := Capture(CaptureRequest{
		RepoRoot: repo, IssuesRoot: ir, Text: hiddenText("a body") + "\nsecond line\n\tindented",
		Severity: SeverityMinor, Category: "bug", Source: "manual-test", Slug: "hidden", FoundDuring: "t",
	})
	if err != nil {
		t.Fatal(err)
	}
	raw := readRaw(t, ir, res.ID)
	assertNoHiddenRune(t, "capture body", raw)
	if !strings.Contains(raw, "\nsecond line\n\tindented") {
		t.Errorf("the body's own line structure was altered:\n%q", raw)
	}
}

func TestWontfixReasonAndDerivedGroundsEncodeHiddenRunes(t *testing.T) {
	repo, ir := ledger(t)
	res, err := Capture(CaptureRequest{RepoRoot: repo, IssuesRoot: ir, Text: "a finding",
		Severity: SeverityMinor, Category: "bug", Source: "manual-test", Slug: "w", FoundDuring: "t"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Wontfix(WontfixRequest{RepoRoot: repo, IssuesRoot: ir, ID: res.ID, Reason: hiddenText("the reason")}); err != nil {
		t.Fatal(err)
	}
	raw := readRaw(t, ir, res.ID)
	fm, body, _ := strings.Cut(raw[4:], "\n---\n")
	assertNoHiddenRune(t, "wontfix_reason", fm)
	assertNoHiddenRune(t, "derived grounds", body)
}

func TestResolveNoteAndGroundsEncodeHiddenRunes(t *testing.T) {
	repo, ir := ledger(t)
	res, err := Capture(CaptureRequest{RepoRoot: repo, IssuesRoot: ir, Text: "a finding",
		Severity: SeverityMinor, Category: "bug", Source: "manual-test", Slug: "r", FoundDuring: "t"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve(ResolveRequest{RepoRoot: repo, IssuesRoot: ir, ID: res.ID, Impact: "fix",
		Resolution: hiddenText("the note"),
		Grounds:    "pursued: " + hiddenText("the conjecture being acted on")}); err != nil {
		t.Fatal(err)
	}
	raw := readRaw(t, ir, res.ID)
	fm, body, _ := strings.Cut(raw[4:], "\n---\n")
	assertNoHiddenRune(t, "resolution", fm)
	assertNoHiddenRune(t, "resolve grounds", body)
}

// TestWontfixDerivedGroundsRefuseAControlCharacter is iss-2608301244450106: the
// grounds a wontfix derives from its reason go through the SAME control-
// character check supplied grounds do (grounds.ValidateText's), so the refusal
// arrives at the grounds boundary as a grounds refusal — not later, at the
// frontmatter serialiser, under the ledger lock. The substance floor stays off
// for a derived value: a terse reason is still a legal wontfix.
func TestWontfixDerivedGroundsRefuseAControlCharacter(t *testing.T) {
	repo, ir := ledger(t)
	res, err := Capture(CaptureRequest{RepoRoot: repo, IssuesRoot: ir, Text: "a finding",
		Severity: SeverityMinor, Category: "bug", Source: "manual-test", Slug: "c", FoundDuring: "t"})
	if err != nil {
		t.Fatal(err)
	}
	before := readRaw(t, ir, res.ID)
	_, err = Wontfix(WontfixRequest{RepoRoot: repo, IssuesRoot: ir, ID: res.ID, Reason: "dup\x0bof another"})
	if !errors.Is(err, ErrGroundsRefused) || !strings.Contains(err.Error(), "U+000B") {
		t.Fatalf("want a grounds refusal naming U+000B, got %v", err)
	}
	if after := readRaw(t, ir, res.ID); after != before {
		t.Fatal("a refused wontfix changed the record")
	}
	// The floor is not applied to a derived value: a one-word reason stands.
	if _, err := Wontfix(WontfixRequest{RepoRoot: repo, IssuesRoot: ir, ID: res.ID, Reason: "duplicate"}); err != nil {
		t.Fatalf("a terse wontfix reason must still be accepted: %v", err)
	}
}
