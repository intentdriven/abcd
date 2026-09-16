package release

import (
	"testing"
)

// A planned intent with one closed spec and one open spec is the correct steady
// state of a partial delivery, not a stale record: the cut must not refuse.
func TestEmitDoesNotRefuseAPartiallyDeliveredIntent(t *testing.T) {
	r := releasedRepo(t)
	r.Record(shippedDir+"itd-73-x.md", "itd-73", "additive")
	r.Write(plannedDir+"itd-94-gate.md", "---\nid: itd-94\nkind: standalone\nspec_id: spc-9\n---\n# gate\n")
	r.Write(specsClosed+"spc-9-gate.md", "---\nid: spc-9\nslug: gate\nintent: itd-94\n---\n# spc-9\n")
	r.Write(specsOpen+"spc-10-rest.md", "---\nid: spc-10\nslug: rest\nintent: itd-94\n---\n# spc-10\n")
	r.Commit("a partial delivery in flight")

	cut := emit(t, r)
	for _, ref := range cut.Refusals {
		if ref.Kind == RefusalStaleIntent {
			t.Fatalf("itd-94 still has an open spec (spc-10); the cut must not refuse: %q %v", ref.Reason, ref.Records)
		}
	}
}
