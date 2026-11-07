package lint

import (
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// The pending-reference half of prose_citation_resolves (iss-2611031731385043):
// a record id the tree does not resolve, but that names a record on an open
// pull request's branch on origin, is a reference to a record in review. It is
// reported and never refused, so an author cites the ruling by id rather than
// in words; the release cut is where it must have landed.

const (
	pendingID     = "dcn-2611031615094092"
	pendingPath   = ".abcd/work/decisions/dcn-2611031615094092-the-run-builds-on-subscriptions.md"
	pendingBranch = "records-priorities-1103"
)

// withPendingLookup registers fn for one test and restores what was there.
func withPendingLookup(t *testing.T, fn PendingLookup) {
	t.Helper()
	prev := pendingLookup
	SetPendingLookup(fn)
	t.Cleanup(func() { SetPendingLookup(prev) })
}

// onPullRequest is a lookup that finds pendingID on #1381 and records every
// id it was asked about, once per call.
func onPullRequest(asked *[][]string) PendingLookup {
	return func(_ string, ids []string) (map[string]PendingRecord, string) {
		got := append([]string(nil), ids...)
		sort.Strings(got)
		*asked = append(*asked, got)
		found := map[string]PendingRecord{}
		for _, id := range ids {
			if id == pendingID {
				found[id] = PendingRecord{ID: id, Path: pendingPath, PR: 1381, Branch: pendingBranch}
			}
		}
		return found, ""
	}
}

// citingIssue is the decision that failed the handover's preflight, recast as
// an issue in the fixture's ledger: it cites the ruling by id, unmarked.
func citingIssue(t *testing.T, root string) string {
	t.Helper()
	name := "iss-2611031720012399-with-both-api-keys-down.md"
	writeProseIssue(t, root, "open", name, "The keys stay a fallback, per "+pendingID+".")
	return filepath.Join(".abcd", "work", "issues", "open", name)
}

// TestProseCitationAcceptsAPendingReference is the behaviour the issue asks
// for: the id names a record on an open pull request's branch, so the gate
// reports it as pending, names the pull request, and refuses nothing.
func TestProseCitationAcceptsAPendingReference(t *testing.T) {
	root := t.TempDir()
	proseCorpus(t, root)
	rel := citingIssue(t, root)
	var asked [][]string
	withPendingLookup(t, onPullRequest(&asked))

	fs, err := Lint(proseCfg(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleProseCitationResolves); n != 0 {
		t.Errorf("a pending reference was refused (%d finding(s)): %+v", n, fs)
	}
	if !hasFinding(fs, rel, ruleProseCitationPending, 5) {
		t.Fatalf("expected the pending reference reported on its line; got %+v", fs)
	}
	for _, f := range fs {
		if f.RuleID != ruleProseCitationPending {
			continue
		}
		if f.Severity != severityInfo {
			t.Errorf("a pending reference is reported at %q, want %q: it must hold nothing up", f.Severity, severityInfo)
		}
		for _, want := range []string{pendingID, "#1381", pendingBranch, pendingPath, "release cut"} {
			if !strings.Contains(f.Message, want) {
				t.Errorf("the pending report does not say %q: %s", want, f.Message)
			}
		}
	}
	if len(asked) != 1 || strings.Join(asked[0], ",") != pendingID {
		t.Errorf("the lookup was asked %v, want once, for %s alone", asked, pendingID)
	}
}

// TestProseCitationRefusesWhatNoOpenPullRequestCarries: the lookup is not a
// licence. An id no open pull request's branch carries is refused as before,
// and the refusal says it was looked for.
func TestProseCitationRefusesWhatNoOpenPullRequestCarries(t *testing.T) {
	root := t.TempDir()
	proseCorpus(t, root)
	name := "iss-2611031720012399-with-both-api-keys-down.md"
	writeProseIssue(t, root, "open", name, "The keys stay a fallback, per dcn-2611039999999999.")
	var asked [][]string
	withPendingLookup(t, onPullRequest(&asked))

	fs, err := Lint(proseCfg(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleProseCitationResolves); n != 1 {
		t.Fatalf("an id nothing carries must be refused once, got %d: %+v", n, fs)
	}
	if countRule(fs, ruleProseCitationPending) != 0 {
		t.Errorf("an id nothing carries was reported as pending: %+v", fs)
	}
	if !messageContains(fs, "No open pull request's branch on origin carries it") {
		t.Errorf("the refusal does not say the open pull requests were looked through: %+v", fs)
	}
}

// TestProseCitationSaysWhyThePendingCheckCouldNotHelp: when the forge or a
// branch could not be read the citation stands refused, the safe direction,
// and the refusal carries the reason so the author can fix the read rather than
// reword the citation.
func TestProseCitationSaysWhyThePendingCheckCouldNotHelp(t *testing.T) {
	root := t.TempDir()
	proseCorpus(t, root)
	citingIssue(t, root)
	withPendingLookup(t, func(string, []string) (map[string]PendingRecord, string) {
		return nil, "the forge client `gh` is not installed, so the open pull requests were not listed"
	})

	fs, err := Lint(proseCfg(), root)
	if err != nil {
		t.Fatal(err)
	}
	if n := countRule(fs, ruleProseCitationResolves); n != 1 {
		t.Fatalf("an unconfirmed pending reference must stay refused, got %d: %+v", n, fs)
	}
	if !messageContains(fs, "`gh` is not installed") {
		t.Errorf("the refusal does not carry why the pending check could not answer: %+v", fs)
	}
}

// TestProseCitationAsksTheForgeOnlyForWhatTheTreeLacks: an ordinary tree,
// whose every citation resolves, never asks; a tree with misses asks once,
// for the distinct ids alone, however many lines cite them.
func TestProseCitationAsksTheForgeOnlyForWhatTheTreeLacks(t *testing.T) {
	root := t.TempDir()
	proseCorpus(t, root)
	writeProseIssue(t, root, "open", "iss-2611031720012398-a-live-citation.md", "Cites iss-42 and itd-73, both live.")
	var asked [][]string
	withPendingLookup(t, onPullRequest(&asked))
	if _, err := Lint(proseCfg(), root); err != nil {
		t.Fatal(err)
	}
	if len(asked) != 0 {
		t.Fatalf("a tree whose citations all resolve asked the forge: %v", asked)
	}

	citingIssue(t, root)
	writeProseIssue(t, root, "open", "iss-2611031720012397-cites-it-twice.md",
		"Again "+pendingID+".\n\nAnd once more, "+pendingID+", beside dcn-2611039999999999.")
	if _, err := Lint(proseCfg(), root); err != nil {
		t.Fatal(err)
	}
	if len(asked) != 1 || strings.Join(asked[0], ",") != "dcn-2611031615094092,dcn-2611039999999999" {
		t.Errorf("the lookup was asked %v, want once, for the two distinct missing ids", asked)
	}
}

// TestProseCitationWriterAcceptsAPendingReference: a writer that asks before
// it writes gets the gate's own answer, so a payload citing a record in review
// is written, by id, rather than refused into a reworded one.
func TestProseCitationWriterAcceptsAPendingReference(t *testing.T) {
	root := t.TempDir()
	proseCorpus(t, root)
	var asked [][]string
	withPendingLookup(t, onPullRequest(&asked))
	rel := filepath.ToSlash(filepath.Join(".abcd", "work", "issues", "open", "iss-2611031720012399-x.md"))

	got, err := UnresolvedProseCitationsInText(proseCfg(), root, rel, "per "+pendingID+", and dcn-2611039999999999\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "dcn-2611039999999999" {
		t.Errorf("the writer refuses %+v, want only the id no branch carries", got)
	}
}
