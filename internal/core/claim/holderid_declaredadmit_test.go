package claim

import (
	"slices"
	"testing"
	"time"
)

// The launch check exempts an admission under an id the reader only
// declares only as a copy of theirs would act on it (iss-2610290809526033):
// its paired login's own newest push records it, and that login pushed its
// first record. One another login wrote is named armed, so the gate refuses
// the cut while it is armed.

// TestTheLaunchCheckExemptsADeclaredIDsAdmissionOnlyWhenVouched: #7 is
// armed and recorded admitted under mayaB, an id octo pairs with maya's
// login, by a release withdrawn then admitted in two commits held under that
// id, which octo takes over, so his own live cut carries the admission.
// Pushed by mal, the launch check names #7 armed, not admitted; pushed by
// maya herself, the admission is exempt.
func TestTheLaunchCheckExemptsADeclaredIDsAdmissionOnlyWhenVouched(t *testing.T) {
	admitted := keptSeven(mayaB)
	admitted.State = WithdrawnAdmitted
	for _, c := range []struct {
		pusher string
		exempt bool
	}{{"mal", false}, {"maya", true}} {
		t.Run(c.pusher, func(t *testing.T) {
			w := newWorld(t)
			f := sevenForge(t)
			withdrawn := admitted
			withdrawn.State = WithdrawnPending
			m := message{Record: "rel-v0.16.0", Holder: mayaB, Copy: "9f2c41d07be35a6c80e1f4b2a7d39c55", ClaimedAt: w.clock(),
				ExpiresAt: w.clock().Add(time.Hour), Withdrawn: []Withdrawal{withdrawn}, AssistedBy: "abcd:0.0.0"}
			f.pusherOf[w.pushTipAt("rel-v0.16.0", m.text(), w.clock())] = c.pusher
			m.Withdrawn = []Withdrawal{admitted}
			w.pushAs(f, "rel-v0.16.0", c.pusher, m.text())
			f.prs[7].method = "SQUASH"
			octo := w.clone("1234+octo@users.noreply.github.com", "")
			declareHolderIDs(t, octo.home, mayaB+" maya\n")
			f.reset()
			if _, err := octo.do(Request{Record: "rel-v0.16.0", TakeOver: true}); err != nil {
				t.Fatal(err)
			}
			if got := actedOn(f, 7); len(got) != 0 {
				t.Fatalf("#7 was acted on: %v; want the admission left as it is", got)
			}
			if e, ok := withdrawalOf(tipEntries(t, w, "rel-v0.16.0"), 7); !ok || e != admitted {
				t.Fatalf("tip records %+v; want octo's take-over to carry the #7 admission", tipEntries(t, w, "rel-v0.16.0"))
			}
			a := armedReading(t, octo)
			if a.Release != "rel-v0.16.0" {
				t.Fatalf("launch reading = %+v; want octo's live cut", a)
			}
			if c.exempt {
				if !slices.ContainsFunc(a.Admitted, func(x Withdrawal) bool { return x == admitted }) || namesArmed(a, 7) {
					t.Fatalf("launch reading = %+v; want #7 exempt as maya's vouched admission", a)
				}
			} else if !namesArmed(a, 7) {
				t.Fatalf("launch reading = %+v; want #7 named armed, not admitted", a)
			}
		})
	}
}
