package landing

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	openIssue     = ".abcd/work/issues/open/iss-1-a.md"
	resolvedIssue = ".abcd/work/issues/resolved/iss-1-a.md"
)

// recordsWorld is a judged head with an open record, and the commit a landing makes.
func recordsWorld(t *testing.T) (*deltaWorld, string) {
	t.Helper()
	w := newDeltaWorld(t)
	w.repo.Write(openIssue, ledgerHead+ledgerBody)
	w.repo.Write("code.go", "package a\n")
	w.repo.Commit("the judged head")
	return w, w.head()
}

// moveRecord is a landing's records commit: the record moved, its head changed.
func moveRecord(t *testing.T, w *deltaWorld) string {
	t.Helper()
	w.repo.Git("rm", "-q", openIssue)
	w.repo.Write(resolvedIssue, "---\nid: iss-1\nresolved_by: abc\n---\n"+ledgerBody)
	w.repo.Commit("chore(record): land lane-1 of run-1")
	return w.head()
}

// TestTheLoopsPairRestsTheHeadOnItsRecordsCommit: the pair read the judged head, and
// the pushed head is that head plus its records.
func TestTheLoopsPairRestsTheHeadOnItsRecordsCommit(t *testing.T) {
	w, judged := recordsWorld(t)
	head := moveRecord(t, w)
	if _, err := RecordLoopReview(w.repo.Root(), deltaDefault, judged, head, fullPair(judged)); err != nil {
		t.Fatal(err)
	}
	if p := readReviewProof(w.repo.Root(), judged, deltaDefault); !p.reviewed || p.note != "" {
		t.Fatalf("the judged head holds the full pair: %+v", p)
	}
	p := readReviewProof(w.repo.Root(), head, deltaDefault)
	if !p.reviewed || !strings.Contains(p.note, short(judged)) {
		t.Fatalf("the pushed head rests on it: %+v", p)
	}
}

// TestALandingWithNoRecordsCommitNeedsOnlyThePair.
func TestALandingWithNoRecordsCommitNeedsOnlyThePair(t *testing.T) {
	w, judged := recordsWorld(t)
	if _, err := RecordLoopReview(w.repo.Root(), deltaDefault, judged, judged, fullPair(judged)); err != nil {
		t.Fatal(err)
	}
	if p := readReviewProof(w.repo.Root(), judged, deltaDefault); !p.reviewed || p.note != "" {
		t.Fatalf("the head holds a full pair: %+v", p)
	}
}

// TestALoopPairThatDoesNotStandWritesNothingForTheHead: a stale pair, one that did not
// pass, and a head that holds more than the records all leave the head with no receipt.
func TestALoopPairThatDoesNotStandWritesNothingForTheHead(t *testing.T) {
	failing := func(sha string) []ReviewReturn {
		return []ReviewReturn{{Name: "ruthless.md", Text: reviewReturn(sha, "FIX FIRST")}, {Name: "security.md", Text: reviewReturn(sha, "APPROVE")}}
	}
	for name, tc := range map[string]struct {
		change  func(t *testing.T, w *deltaWorld)
		returns func(judged, head string) []ReviewReturn
		want    string
	}{
		"a pair naming another head": {
			returns: func(_, head string) []ReviewReturn { return fullPair(head) },
			want:    "does not name the head",
		},
		"a pair that did not pass": {
			returns: func(judged, _ string) []ReviewReturn { return failing(judged) },
			want:    "does not pass",
		},
		"one reviewer": {
			returns: func(judged, _ string) []ReviewReturn { return fullPair(judged)[:1] },
			want:    "at least two",
		},
		"code beside the records": {
			change: func(t *testing.T, w *deltaWorld) { w.repo.Write("code.go", "package b\n"); w.repo.Commit("more") },
			want:   "does not stand on",
		},
		"code in the records commit": {
			change: func(t *testing.T, w *deltaWorld) {
				w.repo.Write("code.go", "package b\n")
				w.repo.Git("add", "-A")
				w.repo.Git("commit", "-q", "--amend", "--no-edit")
			},
			want: "outside the records",
		},
		"a script dressed as a record": {
			change: func(t *testing.T, w *deltaWorld) {
				w.repo.Write(".abcd/work/issues/open/run.sh", "#!/bin/sh\n")
				w.repo.Git("add", "-A")
				w.repo.Git("commit", "-q", "--amend", "--no-edit")
			},
			want: "outside the records",
		},
		"an executable record": {
			change: func(t *testing.T, w *deltaWorld) {
				w.repo.Git("update-index", "--chmod=+x", resolvedIssue)
				w.repo.Git("commit", "-q", "--amend", "--no-edit")
			},
			want: "at mode",
		},
	} {
		t.Run(name, func(t *testing.T) {
			w, judged := recordsWorld(t)
			head := moveRecord(t, w)
			if tc.change != nil {
				tc.change(t, w)
				head = w.head()
			}
			returns := fullPair(judged)
			if tc.returns != nil {
				returns = tc.returns(judged, head)
			}
			_, err := RecordLoopReview(w.repo.Root(), deltaDefault, judged, head, returns)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("refused saying %q, got %v", tc.want, err)
			}
			if p := readReviewProof(w.repo.Root(), head, deltaDefault); p.reviewed {
				t.Fatalf("the head stands: %+v", p)
			}
		})
	}
}

// TestTheGateJudgesTheRecordsReceiptAgain: what a receipt claims earns nothing: a head
// holding more than the records does not arm on one, nor does one resting on a head
// with no full pair.
func TestTheGateJudgesTheRecordsReceiptAgain(t *testing.T) {
	write := func(t *testing.T, w *deltaWorld, head, base string) {
		t.Helper()
		rec := reviewRecord{Commit: head, Reviews: []reviewRecordEntry{{Name: "a", Verdict: "SHIP", SHA256: "x"}, {Name: "b", Verdict: "APPROVE", SHA256: "y"}},
			Delta: &deltaRecord{Base: base, Records: true}}
		raw, err := json.Marshal(rec)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(w.repo.Root(), filepath.FromSlash(ReviewReceiptsRelDir), head)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, raw, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Run("code in the head", func(t *testing.T) {
		w, judged := recordsWorld(t)
		if _, err := RecordReview(w.repo.Root(), judged, fullPair(judged)); err != nil {
			t.Fatal(err)
		}
		w.repo.Write("code.go", "package b\n")
		w.repo.Commit("code")
		write(t, w, w.head(), judged)
		if p := readReviewProof(w.repo.Root(), w.head(), deltaDefault); p.reviewed || p.deltaWhy == "" {
			t.Fatalf("it does not stand: %+v", p)
		}
	})
	t.Run("no pair at the base", func(t *testing.T) {
		w, judged := recordsWorld(t)
		head := moveRecord(t, w)
		write(t, w, head, judged)
		if p := readReviewProof(w.repo.Root(), head, deltaDefault); p.reviewed || !strings.Contains(p.deltaWhy, "no full review pair") {
			t.Fatalf("it does not stand: %+v", p)
		}
	})
}

// TestAStandingReceiptIsLeftAsItIs: a person's full pair for the head is not replaced.
func TestAStandingReceiptIsLeftAsItIs(t *testing.T) {
	w, judged := recordsWorld(t)
	head := moveRecord(t, w)
	if _, err := RecordReview(w.repo.Root(), head, fullPair(head)); err != nil {
		t.Fatal(err)
	}
	if _, err := RecordLoopReview(w.repo.Root(), deltaDefault, judged, head, fullPair(judged)); err != nil {
		t.Fatal(err)
	}
	if p := readReviewProof(w.repo.Root(), head, deltaDefault); !p.reviewed || p.note != "" {
		t.Fatalf("the full pair stands: %+v", p)
	}
}
