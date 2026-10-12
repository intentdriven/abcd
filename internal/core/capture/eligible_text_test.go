package capture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestADrainPlanCarriesNoRawRuneFromTheLedger (iss-2610121516552317): the
// disposition reason the drain builds from ledger text carries no raw control
// or bidi rune, so none reaches `drain --json` or a judgement's note: a
// deferral's tag read with no release tag to judge it by, and the reader's
// refusal of a record whose frontmatter carries U+202E and U+009B, are named
// with the runes masked, and each record keeps its disposition.
func TestADrainPlanCarriesNoRawRuneFromTheLedger(t *testing.T) {
	src := gittest.NewRepo(t)
	src.Commit("root")
	repo := src.Root()
	writeRuleRecord(t, repo, loosenedFields)
	f := drainFixture{t: t, repo: repo, ir: filepath.Join(repo, LedgerRelPath)}
	f.file("iss-2", SeverityMajor, "bug", "rewrite the parser")
	setDeferral(t, f.ir, "iss-2", "v0.1.0‮\u009b")
	bad := filepath.Join(f.ir, "open", "iss-7-broken.md")
	if err := os.WriteFile(bad, []byte("---\nschema_version: \"1‮\u009b\"\nid: \"iss-7\"\n---\n\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := f.plan()
	for _, v := range p.Dispositions {
		if strings.ContainsAny(v.Reason, "‮\u009b") {
			t.Errorf("%s: a raw control rune reaches the disposition reason: %q", v.ID, v.Reason)
		}
	}
	if v := verdictOf(t, p, "iss-2"); v.Outcome != DrainHandBack || v.Rule != RuleDeferred || !strings.Contains(v.Reason, "anchor unknown") ||
		!strings.Contains(v.Reason, "v0.1.0") {
		t.Errorf("the deferral is handed back naming its tag, masked: %+v", v)
	}
	if v := verdictOf(t, p, "iss-7"); v.Outcome != DrainUnreadable || !strings.Contains(v.Reason, "schema_version") {
		t.Errorf("the unreadable record keeps its disposition, naming the refusal: %+v", v)
	}
}
