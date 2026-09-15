package capture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// provenanceFixture captures one issue and plants an intent and a spec record
// so the existence probes have something to find.
func provenanceFixture(t *testing.T) (repo, ir, issID string) {
	t.Helper()
	repo, ir, issID = promoteFixture(t, "an issue that will be resolved with provenance")
	for _, f := range []struct{ rel, id string }{
		{".abcd/development/intents/planned/itd-7-a-planned-intent.md", "itd-7"},
		{".abcd/development/specs/open/spc-3-an-open-spec.md", "spc-3"},
	} {
		abs := filepath.Join(repo, f.rel)
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte("---\nid: "+f.id+"\n---\n\nbody\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return repo, ir, issID
}

// TestResolveWritesResolvedBy is the spc-25 headline: resolve with provenance
// flags writes the resolved_by object — exactly the supplied members, fixed
// member order — in the same atomic transition to resolved/.
func TestResolveWritesResolvedBy(t *testing.T) {
	repo, ir, issID := provenanceFixture(t)

	res, err := Resolve(ResolveRequest{
		Grounds:  testGrounds,
		RepoRoot: repo, IssuesRoot: ir, ID: issID,
		Resolution: "closed with the full trail", Impact: "fix",
		ByIntent: "itd-7", BySpec: "spc-3", ByCommit: "0123abc",
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if res.ToStatus != StateResolved {
		t.Fatalf("ToStatus = %q, want resolved", res.ToStatus)
	}
	if res.ResolvedBy == nil || res.ResolvedBy.Intent != "itd-7" ||
		res.ResolvedBy.Spec != "spc-3" || res.ResolvedBy.Commit != "0123abc" {
		t.Fatalf("result resolved_by = %+v, want all three members", res.ResolvedBy)
	}

	iss := readIssue(t, ir, issID)
	if iss.Status != StateResolved {
		t.Fatalf("issue status = %q, want resolved", iss.Status)
	}
	if iss.ResolvedBy == nil || iss.ResolvedBy.Intent != "itd-7" ||
		iss.ResolvedBy.Spec != "spc-3" || iss.ResolvedBy.Commit != "0123abc" {
		t.Fatalf("parsed-back resolved_by = %+v", iss.ResolvedBy)
	}

	// Fixed member order in the raw frontmatter: intent, spec, commit.
	raw, err := os.ReadFile(iss.Path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	iIdx := strings.Index(content, "intent:")
	sIdx := strings.Index(content, "spec:")
	cIdx := strings.Index(content, "commit:")
	if iIdx < 0 || sIdx < 0 || cIdx < 0 || !(iIdx < sIdx && sIdx < cIdx) {
		t.Fatalf("resolved_by member order not intent,spec,commit:\n%s", content)
	}
}

// TestResolvePartialProvenance: only the supplied members are written — an
// omitted member is omitted, never an empty string.
func TestResolvePartialProvenance(t *testing.T) {
	repo, ir, issID := provenanceFixture(t)
	_, err := Resolve(ResolveRequest{
		Grounds:  testGrounds,
		RepoRoot: repo, IssuesRoot: ir, ID: issID,
		Resolution: "sha only", Impact: "fix", ByCommit: "deadbeefcafe",
	})
	if err != nil {
		t.Fatal(err)
	}
	iss := readIssue(t, ir, issID)
	if iss.ResolvedBy == nil || iss.ResolvedBy.Commit != "deadbeefcafe" {
		t.Fatalf("resolved_by = %+v, want commit only", iss.ResolvedBy)
	}
	if iss.ResolvedBy.Intent != "" || iss.ResolvedBy.Spec != "" {
		t.Fatalf("omitted members must be absent, got %+v", iss.ResolvedBy)
	}
	raw, _ := os.ReadFile(iss.Path)
	if strings.Contains(string(raw), "intent:") || strings.Contains(string(raw), "spec:") {
		t.Fatalf("omitted members leaked into frontmatter:\n%s", raw)
	}
}

// TestResolveProvenanceRefusals: an unknown id or malformed value refuses the
// whole transition — nothing written, the issue stays open.
func TestResolveProvenanceRefusals(t *testing.T) {
	repo, ir, issID := provenanceFixture(t)
	srcPath, _, err := findIssue(ir, issID)
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	for name, req := range map[string]ResolveRequest{
		"unknown intent":   {ByIntent: "itd-99"},
		"malformed intent": {ByIntent: "itd99"},
		"unknown spec":     {BySpec: "spc-99"},
		"malformed spec":   {BySpec: "spec-3"},
		"malformed sha":    {ByCommit: "xyz123"},
		"short sha":        {ByCommit: "abc12"},
	} {
		req.RepoRoot, req.IssuesRoot, req.ID = repo, ir, issID
		req.Resolution, req.Impact = "should refuse", "fix"
		req.Grounds = testGrounds
		if _, err := Resolve(req); err == nil {
			t.Fatalf("%s: Resolve must refuse", name)
		}
		path, status, ferr := findIssue(ir, issID)
		if ferr != nil || status != StateOpen {
			t.Fatalf("%s: issue must stay open, got status=%q err=%v", name, status, ferr)
		}
		after, rerr := os.ReadFile(path)
		if rerr != nil {
			t.Fatal(rerr)
		}
		if string(after) != string(before) {
			t.Fatalf("%s: refused resolve mutated the issue", name)
		}
	}
}

// TestResolveFlaglessWritesNoResolvedBy is the regression guard: a flagless
// resolve stays exactly as today — no resolved_by key at all.
func TestResolveFlaglessWritesNoResolvedBy(t *testing.T) {
	repo, ir, issID := provenanceFixture(t)
	res, err := Resolve(ResolveRequest{
		Grounds:  testGrounds,
		RepoRoot: repo, IssuesRoot: ir, ID: issID, Resolution: "plain", Impact: "fix",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.ResolvedBy != nil {
		t.Fatalf("flagless resolve reported resolved_by: %+v", res.ResolvedBy)
	}
	raw, err := os.ReadFile(filepath.Join(repo, res.Path))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "resolved_by") {
		t.Fatalf("flagless resolve wrote resolved_by:\n%s", raw)
	}
}

// TestFindRecordFileProbe: the shared record-id existence probe finds a record
// in any bucket by the findIssue match rule and reports absence honestly.
func TestFindRecordFileProbe(t *testing.T) {
	repo := t.TempDir()
	for _, rel := range []string{
		".abcd/development/intents/shipped/itd-12-shipped-thing.md",
		".abcd/development/specs/closed/spc-9.md",
	} {
		abs := filepath.Join(repo, rel)
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if rel, ok := findRecordFile(repo, intentStoreRelDirs(), "itd-12"); !ok || !strings.Contains(rel, "shipped") {
		t.Fatalf("probe missed itd-12 in shipped/: %q %v", rel, ok)
	}
	if rel, ok := findRecordFile(repo, specStoreRelDirs(), "spc-9"); !ok || !strings.Contains(rel, "closed") {
		t.Fatalf("probe missed bare spc-9.md in closed/: %q %v", rel, ok)
	}
	if _, ok := findRecordFile(repo, intentStoreRelDirs(), "itd-1"); ok {
		t.Fatalf("probe found an absent id")
	}
	// itd-120 must not match itd-12's prefix rule.
	if _, ok := findRecordFile(repo, intentStoreRelDirs(), "itd-120"); ok {
		t.Fatalf("probe prefix rule over-matched itd-120 against itd-12")
	}
}

// A SHA-256 repo's commits are 64 hex chars; the shape check must accept them
// (iss-356 item 5): the value is a provenance stamp, not a filter, and spc-25's
// own rationale is "must not refuse a legitimate resolution". The sibling
// receipt gate (lint.go receiptShaRe) already accepts {7,64}.
func TestResolveAcceptsASHA256Commit(t *testing.T) {
	repo, ir, issID := provenanceFixture(t)
	sha := strings.Repeat("0123abcd", 8) // 64 hex chars
	_, err := Resolve(ResolveRequest{
		Grounds:  testGrounds,
		RepoRoot: repo, IssuesRoot: ir, ID: issID,
		Resolution: "sha256 repo", Impact: "fix", ByCommit: sha,
	})
	if err != nil {
		t.Fatalf("a 64-hex commit must be accepted: %v", err)
	}
	if iss := readIssue(t, ir, issID); iss.ResolvedBy == nil || iss.ResolvedBy.Commit != sha {
		t.Fatalf("resolved_by = %+v, want the 64-hex sha", iss.ResolvedBy)
	}
}
