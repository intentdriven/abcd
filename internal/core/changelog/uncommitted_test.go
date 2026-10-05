package changelog

import (
	"strings"
	"testing"
)

// TestDeriveRefusesUncommittedRecords pins iss-2610050259118177. The derivation
// reads the release's records at HEAD, so a record move left in the working
// tree — a spec close's intent move, an issue resolve — would be silently left
// out of a cut that otherwise looks plausible. The derivation refuses instead,
// naming every path, while dirt outside the terminal record folders is not its
// concern and a clean tree derives exactly as before.
func TestDeriveRefusesUncommittedRecords(t *testing.T) {
	t.Run("an uncommitted spec close is refused, naming both paths", func(t *testing.T) {
		r := releasedRepo(t)
		r.record(shippedDir+"itd-2-second.md", "itd-2", "fix")
		r.record(plannedDir+"itd-3-third.md", "itd-3", "additive")
		r.commit("ship a fix; itd-3 is planned")
		// The close, left uncommitted: the intent leaves planned/ for shipped/.
		r.record(shippedDir+"itd-3-third.md", "itd-3", "additive")
		r.remove(plannedDir + "itd-3-third.md")
		// And an issue resolve, staged but not committed.
		r.record(resolvedDir+"iss-9-nine.md", "iss-9", "fix")
		r.git("add", resolvedDir+"iss-9-nine.md")

		d, err := Derive(r.root)
		if err != nil {
			t.Fatalf("Derive: %v", err)
		}
		if !d.Refused || d.RefusalKind != RefusalUncommittedRecords {
			t.Fatalf("Refused=%v kind=%q (reason %q), want the uncommitted-records refusal", d.Refused, d.RefusalKind, d.RefusalReason)
		}
		for _, want := range []string{shippedDir + "itd-3-third.md", resolvedDir + "iss-9-nine.md", "commit"} {
			if !strings.Contains(d.RefusalReason, want) {
				t.Errorf("reason does not name %q: %s", want, d.RefusalReason)
			}
		}
		// planned/ is not a terminal folder: the cut never reads it.
		if strings.Contains(d.RefusalReason, plannedDir) {
			t.Errorf("reason names a folder the cut does not read: %s", d.RefusalReason)
		}
		if d.Bumped || d.NextTag != "" {
			t.Errorf("a refused derivation carries a version: %s", d.NextTag)
		}
	})

	t.Run("an uncommitted deletion of a shipped record is refused", func(t *testing.T) {
		r := releasedRepo(t)
		r.record(shippedDir+"itd-2-second.md", "itd-2", "fix")
		r.commit("ship a fix")
		r.remove(shippedDir + "itd-2-second.md")

		d, err := Derive(r.root)
		if err != nil {
			t.Fatalf("Derive: %v", err)
		}
		if d.RefusalKind != RefusalUncommittedRecords || !strings.Contains(d.RefusalReason, shippedDir+"itd-2-second.md") {
			t.Fatalf("kind=%q reason=%q, want the uncommitted-records refusal naming the deleted record", d.RefusalKind, d.RefusalReason)
		}
	})

	t.Run("dirt outside the terminal record folders is not refused", func(t *testing.T) {
		r := releasedRepo(t)
		r.record(shippedDir+"itd-2-second.md", "itd-2", "fix")
		r.commit("ship a fix")
		r.write("README.md", "an edit in progress\n")
		r.write(plannedDir+"itd-4-four.md", "---\nid: itd-4\n---\n# itd-4\n")
		// A non-record file beside the records is not a release line either.
		r.write(shippedDir+"README.md", "notes\n")

		d, err := Derive(r.root)
		if err != nil {
			t.Fatalf("Derive: %v", err)
		}
		if d.Refused {
			t.Fatalf("refused on dirt the cut does not read: %s", d.RefusalReason)
		}
		if d.NextTag != "v0.1.1" {
			t.Errorf("NextTag = %q, want v0.1.1", d.NextTag)
		}
	})

	t.Run("a clean tree derives unchanged", func(t *testing.T) {
		r := releasedRepo(t)
		r.record(shippedDir+"itd-2-second.md", "itd-2", "additive")
		r.commit("ship itd-2")

		d, err := Derive(r.root)
		if err != nil {
			t.Fatalf("Derive: %v", err)
		}
		if d.Refused || d.NextTag != "v0.1.1" {
			t.Fatalf("Refused=%v (%s) NextTag=%q, want v0.1.1", d.Refused, d.RefusalReason, d.NextTag)
		}
	})
}
