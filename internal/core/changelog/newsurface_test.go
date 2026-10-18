package changelog

import (
	"slices"
	"testing"

	"github.com/intentdriven/abcd/internal/core/surface"
)

// TestARecordFoundAtAPathNewSinceTheAnchorIsMarked (iss-2610172148157726): a
// record whose found_at is a path that resolves in HEAD's tree and does not
// resolve in the anchor tag's names a surface the anchor did not have, so a fix
// to it is no fix a reader of the last release could see. Only that shape
// marks it: words (a conceptual location, which capture allows), a path that
// existed at the anchor, one removed since, one at neither end, and one that is
// no repository path at all each leave it unmarked.
func TestARecordFoundAtAPathNewSinceTheAnchorIsMarked(t *testing.T) {
	r := baseRepo(t)
	r.write("old.sh", "old\n")
	r.write("gone.sh", "gone\n")
	r.commit("surfaces at the release")
	r.git("tag", "-f", "v0.1.0")

	r.write("scripts/new.sh", "new\n")
	r.remove("gone.sh")
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
		"iss-2": true,  // a file new since the anchor
		"iss-3": true,  // a directory new since the anchor
		"iss-4": false, // existed at the anchor
		"iss-5": false, // words, not a path
		"iss-6": false, // removed since the anchor: at the anchor, not at HEAD
		"iss-7": false, // at neither end
		"iss-8": false, // not a repository path
		"iss-9": true,  // the same path, unquoted
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
