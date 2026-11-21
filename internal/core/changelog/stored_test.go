package changelog

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/surface"
)

// replaceBlob stands the blob content in for the blob rev:rel names, through
// a refs/replace entry.
func (r *fixtureRepo) replaceBlob(rev, rel, content string) {
	r.t.Helper()
	standIn := filepath.Join(r.t.TempDir(), "stand-in")
	if err := os.WriteFile(standIn, []byte(content), 0o644); err != nil {
		r.t.Fatal(err)
	}
	r.git("replace", r.git("rev-parse", rev+":"+rel), r.git("hash-object", "-w", standIn))
	if got := r.git("cat-file", "blob", rev+":"+rel); got != strings.TrimSpace(content) {
		r.t.Fatalf("the replace ref does not stand in: %s:%s reads %q", rev, rel, got)
	}
}

// cutIDs is the ids of the records a cut adds.
func cutIDs(set RecordSet) []string {
	var ids []string
	for _, rec := range set.Added {
		ids = append(ids, rec.ID)
	}
	return ids
}

// TestGuardSurfaceReadsTheBaselineAsStoredUnderAReplaceRef: a refs/replace
// entry that stands the current snapshot in for the one the release tag
// holds hides the removed command from a read that honours replace refs. The
// guard reads the baseline as stored, so the break is still found.
func TestGuardSurfaceReadsTheBaselineAsStoredUnderAReplaceRef(t *testing.T) {
	baseline := surface.NewSnapshot([]surface.Command{cmdOf("abcd"), cmdOf("abcd ghost")}, nil)
	current := surface.NewSnapshot([]surface.Command{cmdOf("abcd")}, nil)
	r := guardRepo(t, baseline, current, "additive")
	r.git("replace", r.git("rev-parse", "v0.4.0:"+surface.SnapshotPath), r.git("rev-parse", "HEAD:"+surface.SnapshotPath))

	got, err := GuardSurface(r.root, current)
	if err != nil {
		t.Fatalf("GuardSurface: %v", err)
	}
	if got.Status != SurfaceGuardFailed || !strings.Contains(got.Reason, "abcd ghost") {
		t.Fatalf("guard %q (%s), want the stored baseline's abcd ghost reported as removed", got.Status, got.Reason)
	}
}

// TestGuardFindingsReadsTheGradeAsStoredUnderAReplaceRef: a refs/replace
// entry that stands a record graded minor in for a critical finding captured
// this cycle changes nothing the gate judges. The grade is read from the
// record as stored, so the finding still holds the release.
func TestGuardFindingsReadsTheGradeAsStoredUnderAReplaceRef(t *testing.T) {
	r := findingsRepo(t)
	rel := openDir + "iss-2-found.md"
	r.issue(rel, "iss-2", "critical")
	r.commit("capture a finding during this cycle")
	r.replaceBlob("HEAD", rel, "---\nid: \"iss-2\"\nseverity: \"minor\"\n---\n# iss-2\n")

	g, err := GuardFindings(r.root, "v0.1.0")
	if err != nil {
		t.Fatalf("GuardFindings: %v", err)
	}
	if g.Status != FindingGuardFailed || !slices.Equal(findingIDs(g.Unfixed), []string{"iss-2"}) {
		t.Fatalf("status %q, unfixed %v; want the stored critical iss-2 to hold the release", g.Status, findingIDs(g.Unfixed))
	}
}

// TestShippedSinceReadsTheCutAsStored: the cut reads the records, the trees
// and the ancestry it is derived from as stored, so no refs/replace entry or
// graft changes which records it adds or what they declare.
func TestShippedSinceReadsTheCutAsStored(t *testing.T) {
	t.Run("a replaced record keeps its stored impact", func(t *testing.T) {
		r := baseRepo(t)
		rel := resolvedDir + "iss-2-break.md"
		r.record(rel, "iss-2", "breaking")
		r.commit("a breaking fix")
		r.replaceBlob("HEAD", rel, "---\nid: iss-2\nimpact: fix\n---\n# iss-2\n")

		set, err := ShippedSince(r.root, "v0.1.0")
		if err != nil {
			t.Fatalf("ShippedSince: %v", err)
		}
		if !set.DeclaresBreak() {
			t.Fatalf("cut %v declares no break; the stored iss-2 is breaking", cutIDs(set))
		}
	})
	t.Run("a replaced head commit keeps the records it stores", func(t *testing.T) {
		r := baseRepo(t)
		r.record(resolvedDir+"iss-2-fix.md", "iss-2", "fix")
		r.commit("a fix")
		head := r.git("rev-parse", "HEAD")
		r.git("replace", head, r.git("commit-tree", "v0.1.0^{tree}", "-p", "v0.1.0", "-m", "a fix"))

		set, err := ShippedSince(r.root, "v0.1.0")
		if err != nil {
			t.Fatalf("ShippedSince: %v", err)
		}
		if !slices.Equal(cutIDs(set), []string{"iss-2"}) {
			t.Fatalf("cut %v, want the stored head's iss-2", cutIDs(set))
		}
	})
	t.Run("a graft does not put a later tag behind the anchor", func(t *testing.T) {
		r := baseRepo(t)
		anchor := r.git("rev-parse", "v0.1.0")
		later := r.git("commit-tree", r.git("hash-object", "-t", "tree", "-w", "/dev/null"), "-m", "a later release line")
		r.git("tag", "v0.2.0", later)
		r.write(resolvedDir+"iss-2-claimed.md", "---\nid: iss-2\nimpact: breaking\nshipped_in: v0.2.0\n---\n# iss-2\n")
		r.commit("a record naming a release that did not carry it")
		info := filepath.Join(r.root, ".git", "info")
		if err := os.MkdirAll(info, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(info, "grafts"), []byte(anchor+" "+later+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		r.git("-c", "advice.graftFileDeprecated=false", "merge-base", "--is-ancestor", "v0.2.0", "v0.1.0")

		set, err := ShippedSince(r.root, "v0.1.0")
		if err != nil {
			t.Fatalf("ShippedSince: %v", err)
		}
		var found *Record
		for i := range set.Added {
			if set.Added[i].ID == "iss-2" {
				found = &set.Added[i]
			}
		}
		if found == nil || found.ShippedInErr == "" {
			t.Fatalf("cut %v: iss-2 must stay in the cut with its shipped_in reported, since v0.2.0 is no ancestor of the anchor as stored", cutIDs(set))
		}
	})
	t.Run("a replaced anchor does not hide a path new since it", func(t *testing.T) {
		r := baseRepo(t)
		r.write("internal/new.go", "package internal\n")
		r.write(resolvedDir+"iss-2-new.md", "---\nid: iss-2\nimpact: fix\nfound_at: internal/new.go\n---\n# iss-2\n")
		r.commit("a fix to a new file")
		anchor := r.git("rev-parse", "v0.1.0")
		r.git("checkout", "-q", "--detach", anchor)
		r.write("internal/new.go", "package internal\n")
		r.commit("an anchor holding the new file")
		standIn := r.git("commit-tree", "HEAD^{tree}", "-m", "the released state")
		r.git("checkout", "-q", "main")
		r.git("replace", anchor, standIn)
		if got := r.git("ls-tree", "--name-only", "v0.1.0", "--", "internal/new.go"); got != "internal/new.go" {
			t.Fatalf("the replace ref does not stand in: the anchor lists %q", got)
		}

		set, err := ShippedSince(r.root, "v0.1.0")
		if err != nil {
			t.Fatalf("ShippedSince: %v", err)
		}
		for _, rec := range set.Added {
			if rec.ID == "iss-2" {
				if !rec.FoundAtNew {
					t.Fatalf("iss-2's found_at is new since the stored anchor, and was not marked so: %+v", rec)
				}
				return
			}
		}
		t.Fatalf("cut %v, want iss-2", cutIDs(set))
	})
}
