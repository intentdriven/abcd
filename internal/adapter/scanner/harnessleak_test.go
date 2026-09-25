package scanner

// Every session-identifier-shaped fixture below is GENERATED AT RUNTIME, seeded
// per case, so no source line in this repository carries one.
//
// That is the same principle the secret fixtures already keep
// (internal/testsecret, secret-shaped-fixtures-at-runtime), and it is
// load-bearing here for the reason the class exists: this repository scans FULL
// history (gitleaks git, fetch-depth 0) and main cannot be force-pushed, so an
// identifier committed as a literal — even a fabricated one — is in the history
// for good. iss-178 said this detector would be built with "leak shape only, no
// session ids reproduced here", and a literal is also what forces the three
// escapes that would hide it from this class's OWN detector: a line-scoped lint
// waiver, a split literal, and a reserved-documentation host standing in for a
// specimen. Generated, none of the three is needed: the detector can read these
// files and find nothing, because there is nothing here to find.
//
// The positive cases still need a NON-reserved host: hasReservedDocHost would
// otherwise suppress the very detection each one asserts. The host is a generic
// placeholder naming no service; the id is what carried the risk, and the id is
// no longer written down.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/testsecret"
)

// synthSessionID builds a base62 session-id-shaped token at runtime. It asserts
// the token is actually OPAQUE by the detector's own test, so a fixture that
// silently stopped exercising the pattern fails loudly instead of passing a
// weakened assertion. The seed, never the value, is what a failure names.
func synthSessionID(t *testing.T, seed uint64) string {
	t.Helper()
	id := testsecret.Synthetic(seed, 22)
	if !hasOpaqueSessionID("session_" + id) {
		t.Fatalf("generated base62 fixture (seed %d) is not opaque, so it would not exercise the detector", seed)
	}
	return id
}

// synthHexSessionID builds the lower-case-hex spelling a harness also mints.
func synthHexSessionID(t *testing.T, seed uint64) string {
	t.Helper()
	id := testsecret.SyntheticHex(seed, 20)
	if !hasOpaqueSessionID("session_" + id) {
		t.Fatalf("generated hex fixture (seed %d) is not opaque, so it would not exercise the detector", seed)
	}
	return id
}

// synthSessionUUID builds the UUID spelling out of one generated hex run.
func synthSessionUUID(t *testing.T, seed uint64) string {
	t.Helper()
	h := testsecret.SyntheticHex(seed, 32)
	id := h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32]
	if !hasOpaqueSessionID("session/" + id) {
		t.Fatalf("generated UUID fixture (seed %d) is not opaque, so it would not exercise the detector", seed)
	}
	return id
}

// TestHarnessSessionURLDetected: a live session URL is caught wherever it sits,
// which is the point — the harness appends it outside the model's own text, so
// origin cannot be a condition of detection.
func TestHarnessSessionURLDetected(t *testing.T) {
	id := synthSessionID(t, 17)
	for _, line := range []string{
		"https://agent-host.dev/code/session_" + id,
		"See https://agent-host.dev/code/session_" + id + " for the run.",
		"- Session: https://agent-host.dev/s/session-" + id,
		// The UUID and lower-case-hex spellings a harness also mints.
		"https://agent-host.dev/code/session/" + synthSessionUUID(t, 23),
		"https://agent-host.dev/code/session_" + synthHexSessionID(t, 29),
	} {
		if !hasKind(scanLine(line), kindHarnessSessionURL) {
			t.Errorf("session URL not detected in %q", line)
		}
	}
}

// TestHarnessSessionURLSpares an ordinary product URL and a reserved
// documentation host: the first carries no session segment, the second is
// illustrative by convention (the same allowlist-inversion the network patterns
// rest on).
func TestHarnessSessionURLSpares(t *testing.T) {
	for _, line := range []string{
		"The plugin installs from https://agent-host.dev/agent-cli today.",
		"A session URL looks like https://example.invalid/code/session_" + synthSessionID(t, 31),
		"Sessions are stored under .abcd/history/sessions/ per root_commit.",
		// A documentation URL whose slug follows the word "session" with ordinary
		// hyphenated English. Structurally identical to a session link; not one.
		// It is in this repository's own research notes.
		"[Session management and 1M context](https://agent-host.dev/blog/using-agent-session-management-and-1m-context)",
		"https://agent-host.dev/docs/session-configuration-options",
	} {
		if hasKind(scanLine(line), kindHarnessSessionURL) {
			t.Errorf("false positive on %q", line)
		}
	}
}

// TestHarnessSessionURLCoverageBoundary pins the boundary hasOpaqueSessionID
// states (iss-2608281948217899): an all-lower-case id that is not all hex is
// not a session id to this rule, with or without digits, because a
// documentation slug has the same shape. A change that widens the opacity
// test fails here, which is where the slug false positives it buys must be
// weighed.
func TestHarnessSessionURLCoverageBoundary(t *testing.T) {
	for _, id := range []string{"zkqnvqmvxqrtqwyz", "zk3n9q2mvx8rtq", "guide-2026-edition"} {
		if hasOpaqueSessionID("session_" + id) {
			t.Errorf("the lower-case non-hex id %q is inside the rule's coverage; the stated boundary moved", id)
		}
	}
}

// TestHarnessFooterDetected covers the emphasis and emoji spellings the append
// actually lands in — the shapes scripts/check-attribution.sh was widened to
// after each got through it.
func TestHarnessFooterDetected(t *testing.T) {
	for _, line := range []string{
		"Generated with [Some Tool](https://tool.dev)",
		"🤖 Generated with [Some Tool](https://tool.dev)",
		"_Generated with [Some Tool](https://tool.dev)_",
		"🤖 _Generated with [Some Tool](https://tool.dev)_",
		"**Generated by [Some Tool](https://tool.dev)**",
		"  Generated with [Some Tool](https://tool.dev)",
	} {
		if !hasKind(scanLine(line), kindHarnessFooter) {
			t.Errorf("footer not detected in %q", line)
		}
	}
}

// TestHarnessFooterSparesProseAndExamples is the writable-about property the
// shell gate's line anchor already carries: a real footer occupies its own line,
// while prose ABOUT the ban quotes it mid-sentence, a bullet describes it, and an
// example points at a reserved documentation host. All three must survive, or
// this repo cannot document its own rule.
func TestHarnessFooterSparesProseAndExamples(t *testing.T) {
	for _, line := range []string{
		"We ban the \"Generated with [tool](url)\" footer in public text.",
		// A bullet is no longer an excuse on its own — a model-authored footer
		// lands in one — so documentation of the ban points its OWN link at a
		// reserved documentation host, which is what this repository's corpus does.
		"- Generated with [Some Tool](https://example.invalid) is refused.",
		"a single group in one position is exactly the gap that let `🤖 _Generated with [` through",
		"Generated with [Some Tool](https://example.invalid)",
		"The changelog is generated with [the derived cut] rather than by hand.",
	} {
		if hasKind(scanLine(line), kindHarnessFooter) {
			t.Errorf("false positive on %q", line)
		}
	}
}

// TestHarnessLeakPatternsAreInTheCanonicalSet pins the one-definition property:
// the class is declared once and folded into DefaultPatterns, so the four
// store-before-commit consumers, audit, docs-lint and the outbound scan all read
// the same definition rather than drifting copies.
func TestHarnessLeakPatternsAreInTheCanonicalSet(t *testing.T) {
	inDefault := map[string]bool{}
	for _, p := range DefaultPatterns() {
		inDefault[p.Kind] = true
	}
	for _, p := range HarnessLeakPatterns() {
		if !inDefault[p.Kind] {
			t.Errorf("harness-leak kind %q is missing from DefaultPatterns", p.Kind)
		}
		if !IsHarnessLeakKind(p.Kind) {
			t.Errorf("IsHarnessLeakKind does not recognise its own kind %q", p.Kind)
		}
	}
	if len(HarnessLeakPatterns()) != 2 {
		t.Errorf("expected exactly the two leak shapes, got %d", len(HarnessLeakPatterns()))
	}
}

// TestScrubOutboundStripsFooterKeepsTrailer is the outbound criterion: a PR body
// carrying the harness-appended footer comes back carrying only the repo's own
// attribution trailer.
func TestScrubOutboundStripsFooterKeepsTrailer(t *testing.T) {
	body := "Fixes the walk.\n\nAssisted-by: Claude:some-model\n\n🤖 _Generated with [Some Tool](https://tool.dev)_\n"

	got, findings, err := ScrubOutbound(t.TempDir(), body, "pr-body")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "Generated with") {
		t.Errorf("footer survived the scrub: %q", got)
	}
	if !strings.Contains(got, "Assisted-by: Claude:some-model") {
		t.Errorf("the repo's own attribution was lost: %q", got)
	}
	if len(findings) == 0 {
		t.Error("expected the scrub to report what it removed")
	}
}

// TestScrubOutboundCatchesSessionURL: an issue comment carrying a live session
// URL is caught and the URL does not survive.
func TestScrubOutboundCatchesSessionURL(t *testing.T) {
	id := synthSessionID(t, 41)
	comment := "Done — see https://agent-host.dev/code/session_" + id + "\n"

	got, findings, err := ScrubOutbound(t.TempDir(), comment, "issue-comment")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, id) {
		t.Errorf("session URL survived the scrub: %q", got)
	}
	if len(findings) == 0 {
		t.Error("expected the session URL to be reported")
	}
}

// TestScrubOutboundPassesCleanTextUnchanged: an artefact with nothing to strip
// comes back byte-identical, with no finding.
func TestScrubOutboundPassesCleanTextUnchanged(t *testing.T) {
	body := "Fixes the walk over the record stores.\n\nAssisted-by: None\n"

	got, findings, err := ScrubOutbound(t.TempDir(), body, "pr-body")
	if err != nil {
		t.Fatal(err)
	}
	if got != body {
		t.Errorf("clean text was rewritten:\n got %q\nwant %q", got, body)
	}
	if len(findings) != 0 {
		t.Errorf("expected no findings on clean text, got %+v", findings)
	}
}

// TestOutboundPolicyStatesBothHalves pins the policy an autonomous routine's
// prompt carries: the ban on session URLs and harness footers in public text,
// AND the post-create re-read-and-strip, which is the half that matters because
// the append happens outside the model's own output.
func TestOutboundPolicyStatesBothHalves(t *testing.T) {
	policy := strings.ToLower(OutboundPolicy)
	for _, want := range []string{"session url", "footer", "re-read", "strip", "after"} {
		if !strings.Contains(policy, want) {
			t.Errorf("the outbound policy does not state %q:\n%s", want, OutboundPolicy)
		}
	}
}

// The review pass found four ways the class could be evaded or could destroy
// content. Each one is pinned here.

// A footer that lands in a bullet or inside a quoted reply is the MODEL-authored
// half of the class, and it is the half that does not arrive at column 0.
func TestHarnessFooterInListAndQuote(t *testing.T) {
	for _, line := range []string{
		"- 🤖 Generated with [Some Tool](https://tool.dev)",
		"> 🤖 Generated with [Some Tool](https://tool.dev)",
		"  * Generated by [Some Tool](https://tool.dev)",
		"1. Generated with [Some Tool](https://tool.dev)",
	} {
		if !hasKind(scanLine(line), kindHarnessFooter) {
			t.Errorf("footer not detected in %q", line)
		}
	}
}

// The reserved-documentation excuse belongs to the footer's OWN link. Reading the
// rest of the line let any later example.com link disarm a genuine footer.
func TestHarnessFooterDocHostIsScopedToItsOwnLink(t *testing.T) {
	line := "🤖 Generated with [Some Tool](https://tool.dev) — docs at https://example.com/x"
	if !hasKind(scanLine(line), kindHarnessFooter) {
		t.Errorf("an unrelated documentation link disarmed a real footer: %q", line)
	}
}

// A benign leftmost candidate must not disarm the pattern for the rest of the
// line — the shape that made the lint gate weaker than the scanner it shares a
// definition with.
func TestHarnessSessionURLSecondMatchOnALine(t *testing.T) {
	line := "docs https://agent-host.dev/blog/using-agent-session-management-and-1m " +
		"and run https://agent-host.dev/code/session_" + synthSessionID(t, 53)
	if !hasKind(scanLine(line), kindHarnessSessionURL) {
		t.Errorf("a skipped leftmost candidate hid a real session URL: %q", line)
	}
}

// The query-parameter and host:port spellings leak exactly as much as the path
// spelling.
func TestHarnessSessionURLQueryAndPortSpellings(t *testing.T) {
	id := synthSessionID(t, 61)
	for _, line := range []string{
		"https://agent-host.dev/code?session_id=" + id,
		"https://agent-host.dev:8443/code/session_" + id,
	} {
		if !hasKind(scanLine(line), kindHarnessSessionURL) {
			t.Errorf("session URL not detected in %q", line)
		}
	}
}

// A degraded per-repo scanner config must REFUSE, not sanitise with a silently
// weakened pattern set — the contract every sibling write-time redactor keeps.
func TestScrubOutboundRefusesADegradedScanner(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".abcd", "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".abcd", "config", "pii.json"),
		[]byte("{ not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, _, err := ScrubOutbound(root, "Ordinary text.\n", "pr-body")
	if err == nil {
		t.Fatalf("expected a refusal on a degraded config; got %q", got)
	}
	if got != "" {
		t.Errorf("a refusal must hand back nothing to post; got %q", got)
	}
}

// A session URL inside a sentence must not take the sentence with it. Dropping
// the line returned an EMPTY artefact for a one-line comment, which the routine
// would then post.
func TestScrubOutboundKeepsTheSentenceAroundASessionURL(t *testing.T) {
	id := synthSessionID(t, 71)
	body := "Fixes the walk. Session: https://agent-host.dev/code/session_" + id + " — see notes.\n"

	got, findings, err := ScrubOutbound(t.TempDir(), body, "issue-comment")
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) == 0 {
		t.Fatal("expected the session URL to be reported")
	}
	if strings.Contains(got, id) {
		t.Errorf("session URL survived: %q", got)
	}
	if !strings.Contains(got, "Fixes the walk.") || !strings.Contains(got, "see notes.") {
		t.Errorf("the artefact's own content was destroyed: %q", got)
	}
}
