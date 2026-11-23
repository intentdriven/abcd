package loop

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The close ingests the auditor's verdict as the bytes the loop recorded it
// from: a verdict whose rationale changed after the round recorded it is
// refused, with no records commit made and the lane's intent left planned.
func TestTheCloseIngestsOnlyTheVerdictTheRoundRecorded(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.validated(t)
	f.step(t)
	v, ok := passingAuditor(l)
	if !ok || !sha256Hex(v.ReturnSHA256) {
		t.Fatalf("the round records the verdict's digest: %+v", v)
	}
	path := abs(f.repo.Root(), v.Return)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	changed := bytes.Replace(raw, []byte(`"rationale":"read the delivery"`), []byte(`"rationale":"words the auditor never wrote"`), 1)
	if bytes.Equal(changed, raw) {
		t.Fatalf("premise: the verdict carries the fixture's rationale:\n%s", raw)
	}
	if err := os.WriteFile(path, changed, 0o600); err != nil {
		t.Fatal(err)
	}
	_, stepErr := advance(f.repo.Root(), f.runID, f.stages, Options{})
	if stepErr == nil {
		t.Fatal("the records step went through on a changed verdict")
	}
	if r := mustRefusal(t, stepErr); !strings.Contains(r.Reason, "is not the return the loop recorded") {
		t.Fatalf("the refusal names the changed verdict: %+v", r)
	}
	if got := f.repo.Git("-C", l.Worktree, "rev-parse", "HEAD"); got != l.HeadSHA {
		t.Fatalf("no records commit is made: %s, %s", got, l.HeadSHA)
	}
	if _, err := os.Stat(filepath.Join(l.Worktree, filepath.FromSlash(plannedIntentRel))); err != nil {
		t.Fatalf("the lane's intent stays planned: %v", err)
	}

	// Put back as it was handed back, the verdict ingests.
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := advance(f.repo.Root(), f.runID, f.stages, Options{}); err != nil {
		if r, ok := AsRefusal(err); ok && strings.Contains(r.Reason, "is not the return the loop recorded") {
			t.Fatalf("the verdict as recorded is refused: %+v", r)
		}
	}
}

// A state of version 31 carrying a digest of a validator's return that is not a
// SHA-256 in hex is refused as one the loop never writes, and one of version 30
// carrying any digest as one its version never wrote.
func TestAReturnDigestIsReadOnlyAsTheLoopWritesIt(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.validated(t)
	v, _ := passingAuditor(l)
	path := runStatePath(f.repo.Root(), f.runID)
	good, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for name, c := range map[string]struct {
		data   []byte
		reason string
	}{
		"a digest that is no hex": {bytes.Replace(good, []byte(v.ReturnSHA256), []byte(strings.Repeat("Z", 64)), 1), "not a SHA-256 in lower-case hex"},
		"a digest at version 30":  {bytes.Replace(good, []byte(`"schema_version": 31,`), []byte(`"schema_version": 30,`), 1), "`return_sha256`"},
	} {
		t.Run(name, func(t *testing.T) {
			if bytes.Equal(c.data, good) {
				t.Fatal("premise: the state changed")
			}
			if err := os.WriteFile(path, c.data, 0o600); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.WriteFile(path, good, 0o600) })
			_, err := ReadState(f.repo.Root(), f.runID)
			if r, ok := AsRefusal(err); !ok || !strings.Contains(r.Reason, c.reason) {
				t.Fatalf("the read refuses naming %q: %v", c.reason, err)
			}
		})
	}
}
