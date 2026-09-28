package ahoy

import "testing"

// TestUnkindedWriteIsExplainedAsUnexplained is iss-2609260057112822: a write
// that carries no kind, or a kind the table has no entry for, must reach the
// person as a write the summary cannot describe, never as another kind's
// explanation (the old fallback borrowed the scanner hint's, a kind itd-63's
// explain-then-install step retired) and never not at all.
func TestUnkindedWriteIsExplainedAsUnexplained(t *testing.T) {
	r := &InstallResult{
		Writes:     []string{"/x/kinded", "/x/unknown-kind", "/x/no-kind"},
		writeKinds: []writeKind{writeRules, writeKind("no-such-kind")},
	}
	r.explain()
	byRef := map[string]SummaryItem{}
	for _, it := range r.Summary {
		for _, ref := range it.Refs {
			byRef[ref] = it
		}
	}
	if byRef["/x/kinded"].What != writeKindHelp[writeRules].What {
		t.Errorf("a kinded write lost its explanation: %+v", byRef["/x/kinded"])
	}
	for _, w := range []string{"/x/unknown-kind", "/x/no-kind"} {
		it, ok := byRef[w]
		if !ok {
			t.Errorf("write %q is missing from the summary", w)
			continue
		}
		for k, help := range writeKindHelp {
			if it.What == help.What {
				t.Errorf("write %q borrows the %s explanation: %+v", w, k, it)
			}
		}
		if it.What != unexplainedWriteHelp.What {
			t.Errorf("write %q is not reported as unexplained: %+v", w, it)
		}
	}
	assertPlainItem(t, unexplainedWriteHelp)
}
