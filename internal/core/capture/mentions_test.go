package capture

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// The advisory listing's whole job is to tell an operator that an open record
// may already be fixed on the default branch, WITHOUT deciding it for them. So
// every test here asserts two things at once: which rows appear, and that the
// ledger is untouched by the walk.

// mentionsFixture builds a repository whose default branch carries one commit of
// each evidence shape the walk has to tell apart.
func mentionsFixture(t *testing.T) *gittest.Repo {
	t.Helper()
	r := gittest.NewRepo(t)
	for _, id := range []string{"iss-1", "iss-2", "iss-3", "iss-4", "iss-5"} {
		r.Write(".abcd/work/issues/open/"+id+"-a-fixture.md",
			"---\nschema_version: 1\nid: "+id+"\nslug: a-fixture\nseverity: minor\n"+
				"category: bug\nsource: user-observation\nfound_during: t\n---\n\nA fixture.\n")
	}
	// The capture commit itself names the ids it files. That mention is
	// PROVENANCE, not evidence of a fix, and it is the commonest mention in a
	// ledger's history — 76 of this repository's 165 mentioned-open records had
	// no other evidence at all.
	r.Commit("capture: file iss-1, iss-2, iss-3, iss-4 and iss-5")

	// Tree evidence: a commit that changed code and named the record.
	r.Write("cmd/thing/main.go", "package main\n\nfunc main() {}\n")
	r.Commit("fix: the parser hole iss-1 describes")

	// Record evidence: a mention in a commit that touched only the record tiers.
	r.Write(".abcd/work/CONTEXT.md", "orientation\n")
	r.Commit("docs: narrate what iss-2 asked for")

	// Declared as touched-but-not-fixed. `Refs:` is the operator SAYING this is
	// not a fix, so the listing must take them at their word and stay quiet.
	r.Write("cmd/thing/other.go", "package main\n")
	r.Commit("refactor: tidy the ground around iss-3\n\nRefs: iss-3")

	// Declared as fixed, yet the record is still in open/. Strongest evidence
	// there is: somebody said so in the commit and the ledger never moved.
	r.Write("cmd/thing/third.go", "package main\n")
	r.Commit("fix: close the hole\n\nResolves: iss-4")

	// iss-5 is never mentioned again: an open record with no evidence at all.
	return r
}

func rowByID(t *testing.T, res MentionsResult, id string) (MentionRow, bool) {
	t.Helper()
	for _, row := range res.Rows {
		if row.ID == id {
			return row, true
		}
	}
	return MentionRow{}, false
}

func TestMentionsListsOpenRecordsNamedOnTheDefaultBranch(t *testing.T) {
	r := mentionsFixture(t)
	res, err := Mentions(MentionsRequest{RepoRoot: r.Root()})
	if err != nil {
		t.Fatalf("Mentions: %v", err)
	}
	if res.Ref == "" {
		t.Fatalf("the result names no ref; it must say which history it walked: %+v", res)
	}
	if res.OpenRecords != 5 {
		t.Fatalf("OpenRecords = %d, want 5", res.OpenRecords)
	}

	want := map[string]string{"iss-1": StrengthTree, "iss-2": StrengthRecord, "iss-4": StrengthResolves}
	for id, strength := range want {
		row, ok := rowByID(t, res, id)
		if !ok {
			t.Fatalf("%s is named by a default-branch commit and is not listed: %+v", id, res.Rows)
		}
		if row.Strength != strength {
			t.Errorf("%s strength = %q, want %q (%+v)", id, row.Strength, strength, row.Evidence)
		}
		if len(row.Evidence) == 0 {
			t.Errorf("%s is listed with no evidence; a row nobody can check is an accusation", id)
		}
	}
	// The two silences, each for its own reason.
	if _, ok := rowByID(t, res, "iss-3"); ok {
		t.Errorf("iss-3 is declared `Refs:` — touched, not fixed — and must not be listed as possibly resolved")
	}
	if _, ok := rowByID(t, res, "iss-5"); ok {
		t.Errorf("iss-5 is named by no commit at all and must not be listed")
	}
}

// The capture commit is the noisiest false positive available: every open record
// was filed by a commit that may name it. A record whose ONLY mention is the
// commit that added its own file is not evidence of anything.
func TestMentionsIgnoresTheCommitThatFiledTheRecord(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write(".abcd/work/issues/open/iss-9-a-fixture.md",
		"---\nschema_version: 1\nid: iss-9\nslug: a-fixture\nseverity: minor\n"+
			"category: bug\nsource: user-observation\nfound_during: t\n---\n\nA fixture.\n")
	r.Commit("capture: iss-9, a finding from the sweep")
	res, err := Mentions(MentionsRequest{RepoRoot: r.Root()})
	if err != nil {
		t.Fatalf("Mentions: %v", err)
	}
	if _, ok := rowByID(t, res, "iss-9"); ok {
		t.Fatalf("the commit that FILED iss-9 was read as evidence that it is fixed: %+v", res.Rows)
	}
}

// The listing reads; it never resolves. This is the non-negotiable half of the
// feature, so it is asserted directly rather than inferred from the absence of a
// write call.
func TestMentionsNeverMovesARecord(t *testing.T) {
	r := mentionsFixture(t)
	before := r.Git("status", "--porcelain")
	if _, err := Mentions(MentionsRequest{RepoRoot: r.Root()}); err != nil {
		t.Fatalf("Mentions: %v", err)
	}
	if after := r.Git("status", "--porcelain"); after != before {
		t.Fatalf("the walk dirtied the tree: %q -> %q", before, after)
	}
	for _, id := range []string{"iss-1", "iss-2", "iss-3", "iss-4", "iss-5"} {
		path := filepath.Join(r.Root(), ".abcd", "work", "issues", "open", id+"-a-fixture.md")
		matches, err := filepath.Glob(path)
		if err != nil {
			t.Fatalf("glob: %v", err)
		}
		// The result is the assertion: Glob returns an empty slice and a nil error
		// for a path that is gone, so discarding it asserts nothing at all.
		if len(matches) != 1 {
			t.Errorf("%s is no longer in open/ after the walk (glob %q matched %d)", id, path, len(matches))
		}
	}
	res, err := List(ListRequest{RepoRoot: r.Root(), State: StateOpen})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(res.Issues) != 5 {
		t.Fatalf("open records after the walk = %d, want 5 — the listing resolved something", len(res.Issues))
	}
}

// An id inside a longer token is not a mention. The scan runs over free prose,
// where `xiss-1` and `iss-12` both sit next to `iss-1`.
func TestMentionsMatchesWholeIDsOnly(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write(".abcd/work/issues/open/iss-1-a-fixture.md",
		"---\nschema_version: 1\nid: iss-1\nslug: a-fixture\nseverity: minor\n"+
			"category: bug\nsource: user-observation\nfound_during: t\n---\n\nA fixture.\n")
	r.Commit("capture: file the record")
	r.Write("cmd/thing/main.go", "package main\n")
	r.Commit("chore: a commit naming xiss-1 and iss-12 and nothing else")
	res, err := Mentions(MentionsRequest{RepoRoot: r.Root()})
	if err != nil {
		t.Fatalf("Mentions: %v", err)
	}
	if _, ok := rowByID(t, res, "iss-1"); ok {
		t.Fatalf("a substring match listed iss-1: %+v", res.Rows)
	}
}

// A ref the caller names is walked instead of the detected default branch, and a
// ref that does not exist is an error rather than an empty, reassuring listing.
func TestMentionsRefSelection(t *testing.T) {
	r := mentionsFixture(t)
	res, err := Mentions(MentionsRequest{RepoRoot: r.Root(), Ref: "main"})
	if err != nil {
		t.Fatalf("Mentions on main: %v", err)
	}
	if res.Ref != "main" {
		t.Errorf("Ref = %q, want main", res.Ref)
	}
	if _, err := Mentions(MentionsRequest{RepoRoot: r.Root(), Ref: "no-such-branch"}); err == nil {
		t.Fatalf("a missing ref returned a clean empty listing; an operator would read that as 'nothing to do'")
	} else if !strings.Contains(err.Error(), "no-such-branch") {
		t.Errorf("the error does not name the ref that could not be walked: %v", err)
	}
}

// A commit MESSAGE can carry the walk's own record separator, and a message is
// attacker-controlled text. The forged boundary below is shape-perfect — \x1e, 40
// hex, \x1f — and reading it as a commit swaps the real sha for one nobody ever
// made, which `capture resolve --commit` would then accept as provenance. The
// boundary is settled by the commit graph, so the forgery is quoted as what it is:
// part of the real commit's message.
func TestMentionsRefusesAForgedCommitBoundary(t *testing.T) {
	r := gittest.NewRepo(t)
	for _, id := range []string{"iss-1", "iss-2"} {
		r.Write(".abcd/work/issues/open/"+id+"-a-fixture.md",
			"---\nschema_version: 1\nid: "+id+"\nslug: a-fixture\nseverity: minor\n"+
				"category: bug\nsource: user-observation\nfound_during: t\n---\n\nA fixture.\n")
	}
	r.Commit("capture: file iss-1 and iss-2")

	const ghost = "0123456789abcdef0123456789abcdef01234567"
	r.Write("cmd/thing/main.go", "package main\n\nfunc main() {}\n")
	r.Git("add", "-A")
	// --cleanup=verbatim is how an author gets these bytes into a message intact.
	r.Git("commit", "--cleanup=verbatim", "-m",
		"fix: real work\n\n\x1e"+ghost+"\x1f2020-01-01T00:00:00Z\x1fghost body about iss-2\x1f\n\nthis fixes iss-1")
	real := r.Git("rev-parse", "HEAD")

	res, err := Mentions(MentionsRequest{RepoRoot: r.Root()})
	if err != nil {
		t.Fatalf("Mentions: %v", err)
	}
	for _, row := range res.Rows {
		for _, ev := range row.Evidence {
			if ev.Commit == ghost {
				t.Fatalf("%s cites %s — a sha the commit message invented, not a commit: %+v", row.ID, ghost, row.Evidence)
			}
		}
	}
	row, ok := rowByID(t, res, "iss-1")
	if !ok {
		t.Fatalf("iss-1 is named by a real commit and is not listed: %+v", res.Rows)
	}
	if row.Evidence[0].Commit != real {
		t.Fatalf("iss-1 cites %q, want the real commit %q", row.Evidence[0].Commit, real)
	}
}

// A ref name and a path can be the same word. `git log docs` in a repository with
// a `docs/` tree is "ambiguous argument", and this repository has exactly that
// tree — so without the trailing `--` the advisory refuses the histories most
// likely to want it.
func TestMentionsWalksARefNamedAfterATrackedPath(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write(".abcd/work/issues/open/iss-1-a-fixture.md",
		"---\nschema_version: 1\nid: iss-1\nslug: a-fixture\nseverity: minor\n"+
			"category: bug\nsource: user-observation\nfound_during: t\n---\n\nA fixture.\n")
	r.Write("docs/a.md", "a doc\n")
	r.Commit("capture: file iss-1 beside a docs tree")
	r.Write("cmd/thing/main.go", "package main\n")
	r.Commit("fix: the hole iss-1 describes")
	r.Git("branch", "docs")

	res, err := Mentions(MentionsRequest{RepoRoot: r.Root(), Ref: "docs"})
	if err != nil {
		t.Fatalf("a branch named after a tracked directory could not be walked: %v", err)
	}
	if _, ok := rowByID(t, res, "iss-1"); !ok {
		t.Fatalf("iss-1 is named on the walked branch and is not listed: %+v", res.Rows)
	}
}

// The exemplar a row leads with is its STRONGEST evidence, not merely its newest.
// The row an operator must read is the one somebody already declared fixed, and
// ordering by recency alone hid that commit behind a later one that merely
// narrated the record. The order is settled in the core so the text render and
// --json lead with the same commit.
func TestMentionsRanksTheStrongestEvidenceFirst(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write(".abcd/work/issues/open/iss-4-a-fixture.md",
		"---\nschema_version: 1\nid: iss-4\nslug: a-fixture\nseverity: minor\n"+
			"category: bug\nsource: user-observation\nfound_during: t\n---\n\nA fixture.\n")
	r.Commit("capture: file the record")
	r.Write("cmd/thing/main.go", "package main\n")
	r.Commit("fix: close the hole\n\nResolves: iss-4")
	resolves := r.Git("rev-parse", "HEAD")
	r.Write(".abcd/work/CONTEXT.md", "orientation\n")
	r.Commit("docs: narrate what iss-4 asked for")

	res, err := Mentions(MentionsRequest{RepoRoot: r.Root()})
	if err != nil {
		t.Fatalf("Mentions: %v", err)
	}
	row, ok := rowByID(t, res, "iss-4")
	if !ok {
		t.Fatalf("iss-4 is not listed: %+v", res.Rows)
	}
	if row.Evidence[0].Strength != StrengthResolves || row.Evidence[0].Commit != resolves {
		t.Fatalf("Evidence[0] = %+v, want the %q commit %s first (%+v)",
			row.Evidence[0], StrengthResolves, resolves, row.Evidence)
	}
	if len(row.Evidence) != 2 {
		t.Fatalf("evidence = %+v, want both the resolves commit and the docs commit", row.Evidence)
	}
	if row.Evidence[1].Strength != StrengthRecord {
		t.Errorf("the weaker evidence was dropped rather than ranked below: %+v", row.Evidence)
	}
}

// `Refs: iss-1, iss-2` is the conventional trailer shape. Both halves of the rule
// read it — the gate's DECLARE_RE and this twin — so a listing cannot report an
// id the gate has already accepted as declared.
func TestMentionsReadsACommaSeparatedDeclarationList(t *testing.T) {
	r := gittest.NewRepo(t)
	for _, id := range []string{"iss-1", "iss-2"} {
		r.Write(".abcd/work/issues/open/"+id+"-a-fixture.md",
			"---\nschema_version: 1\nid: "+id+"\nslug: a-fixture\nseverity: minor\n"+
				"category: bug\nsource: user-observation\nfound_during: t\n---\n\nA fixture.\n")
	}
	r.Commit("capture: file the records")
	r.Write("cmd/thing/main.go", "package main\n")
	r.Commit("refactor: tidy the ground around iss-1 and iss-2\n\nRefs: iss-1, iss-2")

	res, err := Mentions(MentionsRequest{RepoRoot: r.Root()})
	if err != nil {
		t.Fatalf("Mentions: %v", err)
	}
	for _, id := range []string{"iss-1", "iss-2"} {
		if _, ok := rowByID(t, res, id); ok {
			t.Errorf("%s is declared on a `Refs:` list — touched, not fixed — and must not be listed: %+v", id, res.Rows)
		}
	}
}
