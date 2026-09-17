package scanner

// The outbound-artefact scrub: the primitive an autonomous run calls on every
// pull-request body, issue and comment BEFORE it is posted.
//
// abcd opens no pull requests and owns no forge client, and this does not change
// that (spc-45). It is a primitive, not a posting path: the routine holds the
// forge credentials and the routine calls this on the text it is about to send.
// What abcd owns is the definition of a leak, and this is the one door onto it
// for outward-facing text.
//
// Why a scrub rather than a check. Both leak shapes are appended OUTSIDE the
// model's own output, when the artefact is created — so a rule that only refuses
// text the model composed catches nothing, and a forge keeps the pre-edit
// revision of whatever was posted, which makes "post then fix" no remedy at all.
// The routine therefore re-reads what it created and strips, and that is what
// OutboundPolicy makes every autonomous prompt say.

import (
	"fmt"
	"strings"
)

// OutboundPolicy is the policy block an autonomous routine's prompt carries. It
// is a value rather than prose in a document because the prompt assembly, the
// audit rule's fix hint and this package's own contract must say the same thing:
// a policy that exists in three wordings is a policy with three meanings.
const OutboundPolicy = "Never put a live session URL or a tool's own attribution footer " +
	"(\"generated with/by <tool>\") into public text — a pull-request body, an issue, a comment, " +
	"a commit message or a release note. Disclosure goes in the repository's own trailer " +
	"(Assisted-by: <Vendor>:<model-version>) and nowhere else. " +
	"After creating ANY pull request, issue or comment, re-read what was actually created and strip " +
	"any session URL or attribution footer from it: the harness appends them outside your own output, " +
	"so text that left you clean can arrive dirty, and the forge keeps the pre-edit revision of what " +
	"was posted — which is why the strip has to happen immediately, not at review time."

// ScrubOutbound sanitises one outbound artefact and returns the text to post,
// the findings that were removed, and a fail-closed error if anything blocking
// survived.
//
// It is the ScanText → Redact → residual-rescan shape the store-before-commit
// paths already use (history, memory), with one difference: an attribution
// FOOTER takes its whole line rather than being masked in place. Masking is
// right for a secret, where the surrounding text is the artefact's own content;
// a footer IS the whole line the harness added, and leaving a masked stub behind
// would post the shape of the thing the policy bans. A session URL is masked
// like any other span, because it can sit inside a sentence the artefact meant
// to say.
//
// repoRoot supplies the per-repo scanner config, so a repository that raised a
// severity in .abcd/config/pii.json is honoured here exactly as in Stage-1
// redaction.
func ScrubOutbound(repoRoot, text, label string) (string, []Finding, error) {
	sc, err := New(repoRoot)
	if err != nil {
		return "", nil, err
	}
	// The degraded-config refusal every write-time redactor in this repository
	// makes (capture, memory, intent, history, and the audit rule). New() returns
	// a usable scanner on every degradation path, so an unreadable or unparseable
	// .abcd/config/pii.json silently drops the repo's OWN detectors and leaves the
	// built-in set — and this is the one surface whose output a forge keeps the
	// pre-edit revision of. Sanitising with a weakened set and reporting success
	// is worse here than refusing.
	if degraded, reason := sc.Unavailable(); degraded {
		return "", nil, fmt.Errorf("outbound artefact %q: refusing to sanitise with a degraded scanner config: %s", label, reason)
	}

	findings := sc.ScanText(text, label)
	if len(findings) == 0 {
		return text, nil, nil
	}

	// Stage one: drop the lines a FOOTER occupies. A footer is a whole line by
	// construction — its own pattern refuses a match with prose before it — so
	// removing the line removes exactly the thing that was appended, and leaving a
	// masked stub behind would post the shape the policy bans.
	//
	// A session URL is NOT dropped this way. It can sit mid-sentence in text the
	// artefact genuinely wanted to say, and taking its line then deletes the
	// author's content — in the limit returning an empty artefact, which the
	// routine would post. It is masked in place by Redact below instead.
	drop := map[int]bool{}
	for _, f := range findings {
		if f.Kind == kindHarnessFooter {
			drop[f.Line] = true
		}
	}
	stripped := dropLines(text, drop)

	// Stage two: mask whatever else the scan found in what remains. The rescan is
	// over the STRIPPED text so a finding's line number matches the text Redact is
	// about to rewrite — reusing the first scan's numbers would mask the wrong
	// lines once anything has been removed.
	rest := sc.ScanText(stripped, label)
	redacted, _ := Redact(stripped, rest)

	// Stage three, fail closed. Redact is only stage one by its own contract, and
	// a caller that posts what it could not sanitise is worse than one that
	// refuses: the artefact is public the moment it is created.
	for _, f := range sc.ScanText(redacted, label) {
		if f.Severity == SeverityHardFail || IsHarnessLeakKind(f.Kind) {
			return "", findings, fmt.Errorf("outbound artefact %q still carries a %s after redaction; refusing to hand back text to post", label, f.Kind)
		}
	}
	return redacted, findings, nil
}

// CheckOutbound is the CHECK-direction twin of ScrubOutbound: it reports the
// outbound-policy findings in one artefact and refuses, and it has no way to
// hand back rewritten text at all.
//
// WHY A SECOND DIRECTION, when ScrubOutbound already exists. A scrub is right
// for text a ROUTINE is about to post: the routine owns that text, nobody has
// read it yet, and rewriting it is the remedy. A gate judges text a PERSON
// already wrote — a commit message in a pull request's range, a pull-request
// body — and rewriting that is not a remedy, it is an edit made on the author's
// behalf to something already in the history. So this direction returns no text.
// The absence of a string return is the guarantee: a caller cannot silently
// rewrite an author's commit message through this door, because the door has no
// such outlet.
//
// WHY IT REPORTS ONLY THE HARNESS-LEAK CLASS, where the scrub masks everything
// it finds. Masking more than the policy names is free — the artefact still
// reads and the extra mask costs the routine nothing. REFUSING more than the
// policy names is not free: this runs as a required check over every commit
// message of every pull request, so each extra class is a new way for the gate
// to go red on text that breaks no stated rule (a commit message quoting a
// private address is the live example), and a gate that reds on the innocent is
// a gate somebody switches off. Committed text is judged for the other classes
// by `abcd lint`'s privacy rule and by the record/docs `harness_leak` rule; this
// door judges the two shapes a harness stamps onto public text, which is the
// class OutboundPolicy actually states.
//
// The error is non-nil whenever the artefact is refused, findings or not, so a
// caller that reads only the error still fails closed.
func CheckOutbound(repoRoot, text, label string) ([]Finding, error) {
	sc, err := New(repoRoot)
	if err != nil {
		return nil, err
	}
	// Same degraded-config refusal ScrubOutbound makes, for the same reason and
	// then one more. New() returns a usable scanner on every degradation path,
	// so an unreadable or unparseable .abcd/config/pii.json silently drops the
	// repo's OWN detectors and leaves the built-in set — and a GATE that reports
	// "clean" from a weakened set is worse than one that reports nothing: the
	// green tick is read as "this was checked".
	if degraded, reason := sc.Unavailable(); degraded {
		return nil, fmt.Errorf("outbound artefact %q: refusing to judge with a degraded scanner config: %s", label, reason)
	}

	var leaks []Finding
	for _, f := range sc.ScanText(text, label) {
		if IsHarnessLeakKind(f.Kind) {
			leaks = append(leaks, f)
		}
	}
	if len(leaks) == 0 {
		return nil, nil
	}
	return leaks, fmt.Errorf("outbound artefact %q carries %d outbound-policy violation(s); %s",
		label, len(leaks), OutboundPolicy)
}

// dropLines removes the 1-based line numbers in drop from text, preserving the
// trailing-newline shape of the input (a text ending in "\n" still does).
func dropLines(text string, drop map[int]bool) string {
	if len(drop) == 0 {
		return text
	}
	lines := strings.Split(text, "\n")
	kept := make([]string, 0, len(lines))
	for i, line := range lines {
		if drop[i+1] {
			continue
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}
