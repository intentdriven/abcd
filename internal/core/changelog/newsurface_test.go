package changelog

import (
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/surface"
	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// TestARecordFoundAtAPathNewSinceTheAnchorIsMarked (iss-2610172148157726): a
// record whose found_at is a path that resolves in HEAD's tree and does not
// resolve in the anchor tag's names a surface the anchor did not have, so a fix
// to it is no fix a reader of the last release could see. Only that shape
// marks it: words (a conceptual location, which capture allows), a path that
// existed at the anchor, one removed since, one at neither end, one that is no
// plain repository path, and one renamed since the anchor (a new path for a
// surface the anchor had) each leave it unmarked.
func TestARecordFoundAtAPathNewSinceTheAnchorIsMarked(t *testing.T) {
	r := baseRepo(t)
	r.write("old.sh", "old\n")
	r.write("gone.sh", "gone\n")
	r.write("lib/drain.go", "package lib\n\nfunc Drain() {}\n// a body long enough for git to see the move as a rename\n")
	r.commit("surfaces at the release")
	r.git("tag", "-f", "v0.1.0")

	r.write("scripts/new.sh", "new\n")
	r.remove("gone.sh")
	// A file moved since the anchor: a new path for a surface the anchor had.
	r.write("moved/drain.go", "package lib\n\nfunc Drain() {}\n// a body long enough for git to see the move as a rename\n")
	r.remove("lib/drain.go")
	rec := func(id, foundAt string) {
		r.write(resolvedDir+id+"-x.md", "---\nid: "+id+"\nimpact: fix\nfound_at: \""+foundAt+"\"\n---\n# "+id+"\n")
	}
	rec("iss-2", "scripts/new.sh")
	rec("iss-3", "scripts")
	rec("iss-4", "old.sh")
	rec("iss-5", "the drain's renewal of a lane in progress")
	rec("iss-6", "gone.sh")
	rec("iss-7", "never.sh")
	rec("iss-8", "/scripts/new.sh")
	rec("iss-10", "moved/drain.go")
	rec("iss-11", "moved")
	rec("iss-12", ":(top)scripts/new.sh")
	// A bare value, as YAML allows beside the quoted form capture writes.
	r.write(resolvedDir+"iss-9-x.md", "---\nid: iss-9\nimpact: fix\nfound_at: scripts/new.sh\n---\n# iss-9\n")
	r.commit("fixes since the release")

	set, err := ShippedSince(r.root, "v0.1.0")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, rec := range set.Added {
		got[rec.ID] = rec.FoundAtNew
	}
	for id, want := range map[string]bool{
		"iss-2":  true,  // a file new since the anchor
		"iss-3":  true,  // a directory new since the anchor
		"iss-4":  false, // existed at the anchor
		"iss-5":  false, // words, not a path
		"iss-6":  false, // removed since the anchor: at the anchor, not at HEAD
		"iss-7":  false, // at neither end
		"iss-8":  false, // not a repository path
		"iss-9":  true,  // the same path, unquoted
		"iss-10": false, // renamed since the anchor: a new path, an old surface
		"iss-11": false, // a directory holding a renamed file
		"iss-12": false, // pathspec magic, not a plain path
	} {
		if v, ok := got[id]; !ok {
			t.Errorf("%s is not in the cut", id)
		} else if v != want {
			t.Errorf("%s FoundAtNew = %v; want %v", id, v, want)
		}
	}
}

// TestTheGuardNamesTheCommandsNewSinceTheAnchor (iss-2610172148157726): the
// guard already reads the anchor's surface snapshot, so it names the commands
// the current surface holds that the anchor's did not, in canonical order. A
// composer reads them to know which surfaces the last release lacked, so a fix
// to one of them is worded as what it does rather than as a fix.
func TestTheGuardNamesTheCommandsNewSinceTheAnchor(t *testing.T) {
	baseline := surface.NewSnapshot([]surface.Command{cmdOf("abcd"), cmdOf("abcd peers")}, nil)
	current := surface.NewSnapshot([]surface.Command{cmdOf("abcd"), cmdOf("abcd peers"), cmdOf("abcd status"), cmdOf("abcd announce")}, nil)
	r := guardRepo(t, baseline, current, "additive")

	got, err := GuardSurface(r.root, current)
	if err != nil {
		t.Fatalf("GuardSurface: %v", err)
	}
	if want := []string{"abcd announce", "abcd status"}; !slices.Equal(got.NewCommands, want) {
		t.Errorf("NewCommands = %q; want %q", got.NewCommands, want)
	}
	if got.Status != SurfaceGuardPassed {
		t.Errorf("Status = %q; a command added is no break", got.Status)
	}
}

// TestABloblessCloneDoesNotMarkAPathTheAnchorHad: in a partial clone that
// fetched no blobs, the anchor's copy of a file edited since is not in the
// object store, and a read that needs the blob would take the path for one the
// anchor lacked. Presence is read from the trees, which the clone holds, so a
// fix to that file is not marked new.
func TestABloblessCloneDoesNotMarkAPathTheAnchorHad(t *testing.T) {
	src := baseRepo(t)
	src.write("old.sh", "old\n")
	src.commit("a surface at the release")
	src.git("tag", "-f", "v0.1.0")
	src.write("old.sh", "old, edited since\n")
	src.write(resolvedDir+"iss-2-x.md", "---\nid: iss-2\nimpact: fix\nfound_at: \"old.sh\"\n---\n# iss-2\n")
	src.commit("a fix to it")
	src.git("config", "uploadpack.allowfilter", "true")

	dir := filepath.Join(t.TempDir(), "clone")
	src.git("clone", "-q", "--no-checkout", "--filter=blob:none", "file://"+src.root, dir)
	// Check out HEAD with lazy fetching allowed for this one command, so the
	// clone holds HEAD's blobs and not the anchor's copy of old.sh.
	var env []string
	for _, kv := range gittest.Env(t) {
		if !strings.HasPrefix(kv, "GIT_NO_LAZY_FETCH=") {
			env = append(env, kv)
		}
	}
	co := exec.Command("git", "-C", dir, "checkout", "-q", "HEAD", "--", ".")
	co.Env = env
	if out, err := co.CombinedOutput(); err != nil {
		t.Fatalf("checking out the clone: %v\n%s", err, out)
	}
	if out, err := gitutil.Run(dir, "cat-file", "-e", "v0.1.0:old.sh"); err == nil {
		t.Skipf("the clone holds the anchor's blob (%q), so it is not the partial clone this test needs", out)
	}

	set, err := ShippedSince(dir, "v0.1.0")
	if err != nil {
		t.Fatal(err)
	}
	for _, rec := range set.Added {
		if rec.ID == "iss-2" && rec.FoundAtNew {
			t.Errorf("iss-2 is marked new in a blobless clone, though v0.1.0 holds old.sh")
		}
	}
}
