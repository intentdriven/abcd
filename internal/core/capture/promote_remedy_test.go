package capture

import (
	"errors"
	"strings"
	"testing"
)

// remedySpan returns the text of the CommonMark code span that follows lead in
// msg: a run of N backticks opens it and the next run of EXACTLY N closes it,
// and one space is stripped from each end when both are present. It is the
// reading a renderer, a test or any tool applies, so a remedy is well delimited
// only when this returns the whole command.
func remedySpan(t *testing.T, msg, lead string) string {
	t.Helper()
	i := strings.Index(msg, lead)
	if i < 0 {
		t.Fatalf("message carries no %q: %s", lead, msg)
	}
	rest := msg[i+len(lead):]
	n := 0
	for n < len(rest) && rest[n] == '`' {
		n++
	}
	if n == 0 {
		t.Fatalf("no code span follows %q: %s", lead, msg)
	}
	body := rest[n:]
	for j := 0; j < len(body); {
		if body[j] != '`' {
			j++
			continue
		}
		k := j
		for k < len(body) && body[k] == '`' {
			k++
		}
		if k-j == n {
			span := body[:j]
			if len(span) >= 2 && span[0] == ' ' && span[len(span)-1] == ' ' {
				span = span[1 : len(span)-1]
			}
			return span
		}
		j = k
	}
	t.Fatalf("the code span after %q is never closed: %s", lead, msg)
	return ""
}

// TestPromoteOrphanRemedySpanIsTheWholeCommand is iss-2609020154474224: the
// remedy carries the promotion's grounds, which are free prose, and a backtick
// in them closed a single-backtick delimiter early, so whatever read the text
// between the delimiters got a truncated command. The delimiter is now one the
// argument cannot close.
func TestPromoteOrphanRemedySpanIsTheWholeCommand(t *testing.T) {
	repo, ir, issID := promoteFixture(t, "the stamp will fail after the mint")
	stampWriteHook = func(string, []byte) error { return errors.New("simulated unwritable ledger") }
	t.Cleanup(func() { stampWriteHook = nil })
	const g = "pursued: we expect the `--flag` spelling and a ``double`` run to survive in the remedy"
	_, err := Promote(PromoteRequest{Grounds: g, RepoRoot: repo, IssuesRoot: ir, ID: issID})
	if err == nil {
		t.Fatal("stamp into an unwritable ledger must fail")
	}
	span := remedySpan(t, err.Error(), "complete the link with ")
	words := shellWords(t, span)
	if len(words) != 8 || words[6] != "--grounds" || words[7] != g {
		t.Fatalf("the delimited remedy is not the whole command: %q (from %s)", words, err)
	}
}

// TestPromoteOrphanRemedyWithoutGroundsNamesNoGrounds: grounds are optional on
// the issue route, so a promotion given none can fail its stamp too. The remedy
// then carries no --grounds, and the report is an error, never a crash on the
// absent value.
func TestPromoteOrphanRemedyWithoutGroundsNamesNoGrounds(t *testing.T) {
	repo, ir, issID := promoteFixture(t, "the stamp will fail after the mint")
	stampWriteHook = func(string, []byte) error { return errors.New("simulated unwritable ledger") }
	t.Cleanup(func() { stampWriteHook = nil })
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("a failed promotion given no grounds panicked composing its remedy: %v", r)
			}
		}()
		_, err = Promote(PromoteRequest{RepoRoot: repo, IssuesRoot: ir, ID: issID})
	}()
	if err == nil {
		t.Fatal("stamp into an unwritable ledger must fail")
	}
	span := remedySpan(t, err.Error(), "complete the link with ")
	orphan := soleDraftID(t, repo)
	if span != "abcd capture promote "+issID+" --intent "+orphan {
		t.Fatalf("remedy = %q, want the link command with no --grounds", span)
	}
}

// TestPromoteLostRaceRemedyNamesTheDuplicateDraft is iss-258: a promotion that
// loses the race to a concurrent promotion of the same issue has minted a
// duplicate draft, and the link remedy would itself be refused as
// already-promoted. The report names the promotion that won and says to delete
// the duplicate instead.
func TestPromoteLostRaceRemedyNamesTheDuplicateDraft(t *testing.T) {
	repo, ir, issID := promoteFixture(t, "two promotions race")
	var winner string
	beforePromoteStampHook = func() {
		beforePromoteStampHook = nil
		res, err := Promote(PromoteRequest{Grounds: testGrounds, RepoRoot: repo, IssuesRoot: ir, ID: issID})
		if err != nil {
			t.Fatalf("the winning promotion: %v", err)
		}
		winner = res.IntentID
	}
	t.Cleanup(func() { beforePromoteStampHook = nil })

	_, err := Promote(PromoteRequest{Grounds: testGrounds, RepoRoot: repo, IssuesRoot: ir, ID: issID})
	if err == nil {
		t.Fatal("the losing promotion must fail")
	}
	if !errors.Is(err, ErrAlreadyPromoted) {
		t.Fatalf("want ErrAlreadyPromoted, got %v", err)
	}
	msg := err.Error()
	if strings.Contains(msg, "complete the link") {
		t.Fatalf("the lost race still advises the link, which would be refused: %s", msg)
	}
	if !strings.Contains(msg, winner) || !strings.Contains(msg, "delete the duplicate draft") {
		t.Fatalf("the report must name the winning intent %s and say to delete the duplicate: %s", winner, msg)
	}
}
