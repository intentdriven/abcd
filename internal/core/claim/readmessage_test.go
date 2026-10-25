package claim

import (
	"slices"
	"strings"
	"testing"
	"time"
)

// TestReadReservationMessageReadsWhatTheWriterWrites: a reservation, a plain
// tombstone, a hand-back tombstone and a release's reservation, each written
// by the package's own writer, read back through the exported reader with
// every field the drain's reader keeps, the record taken from the message.
func TestReadReservationMessageReadsWhatTheWriterWrites(t *testing.T) {
	const blob = "0123456789abcdef0123456789abcdef01234567"
	res := message{
		Record: "iss-2609020716570699", Holder: "octo", Copy: "9f2c41d07be35a6c80e1f4b2a7d39c55",
		Footprint: []string{"internal/core/claim/claim.go"}, ClaimedAt: t0, ExpiresAt: t0.Add(2 * time.Hour),
		AssistedBy: "abcd:v0.9.0",
	}
	plain := res
	plain.ExpiresAt, plain.ReleasedAt = t0.Add(time.Hour), t0.Add(time.Hour)
	handBack := plain
	handBack.HandedBack, handBack.RecordBlob = HandBackDesignFinding, blob
	release := res
	release.Record, release.Footprint = "rel-v0.16.0", nil
	for name, m := range map[string]message{"a reservation": res, "a plain tombstone": plain, "a hand-back": handBack, "a release's reservation": release} {
		got, err := ReadReservationMessage(m.text())
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got.Record != m.Record || got.Holder != m.Holder || got.Copy != m.Copy || !slices.Equal(got.Footprint, m.Footprint) ||
			!got.ClaimedAt.Equal(m.ClaimedAt) || !got.ExpiresAt.Equal(m.ExpiresAt) || !got.ReleasedAt.Equal(m.ReleasedAt) ||
			got.HandedBack != m.HandedBack || got.RecordBlob != m.RecordBlob || got.AssistedBy != m.AssistedBy {
			t.Fatalf("%s reads back as %+v; want %+v", name, got, m)
		}
		if got.Released() != m.released() {
			t.Fatalf("%s: Released() = %v; want %v", name, got.Released(), m.released())
		}
		subject, _, _ := strings.Cut(m.text(), "\n\n")
		if got.Lead != subject || len(got.Other) != 0 {
			t.Fatalf("%s: lead %q, other %q; want the subject %q and nothing else", name, got.Lead, got.Other, subject)
		}
	}
}

// TestReadReservationMessageRefusesWhatTheDrainCannotRead: every message the
// drain's reader reads as unreadable is an error here, with the drain's own
// reason, and a key the reader does not know is ignored, as the drain ignores
// it, but handed back in Other with any text outside the trailers in Lead.
func TestReadReservationMessageRefusesWhatTheDrainCannotRead(t *testing.T) {
	plain := message{
		Record: "iss-5", Holder: "octo", Copy: "9f2c41d07be35a6c80e1f4b2a7d39c55",
		ClaimedAt: t0, ExpiresAt: t0.Add(time.Hour), ReleasedAt: t0.Add(time.Hour),
	}.text()
	reservation := message{Record: "iss-5", Holder: "octo", Copy: "9f2c41d07be35a6c80e1f4b2a7d39c55", ClaimedAt: t0, ExpiresAt: t0.Add(time.Hour)}.text()
	const blob = "\nAbcd-Record: 0123456789abcdef0123456789abcdef01234567"
	for name, tc := range map[string]struct{ text, why string }{
		"a hand-back kind outside the vocabulary": {plain + "Abcd-Handed-Back: needs-decision" + blob + "\n", reasonHandBack},
		"a hand-back without its record blob":     {plain + "Abcd-Handed-Back: " + HandBackStuck + "\n", reasonHandBack},
		"a hand-back on a reservation":            {reservation + "Abcd-Handed-Back: " + HandBackStuck + blob + "\n", reasonHandBack},
		"no record":                               {strings.Replace(plain, "Abcd-Reservation: iss-5\n", "", 1), reasonRecord},
		"two records":                             {strings.Replace(plain, "Abcd-Reservation: iss-5\n", "Abcd-Reservation: iss-5\nAbcd-Reservation: iss-6\n", 1), reasonRecord},
		"a holder that is no username":            {strings.Replace(plain, "Abcd-Holder: octo", "Abcd-Holder: not a user", 1), reasonHolder},
		"a copy in another form":                  {strings.Replace(plain, "Abcd-Copy: 9f2c41d07be35a6c80e1f4b2a7d39c55", "Abcd-Copy: laptop", 1), reasonCopy},
		"a time that does not read":               {strings.Replace(plain, "Abcd-Released-At: 2026-10-03T15:00:00Z", "Abcd-Released-At: soon", 1), reasonTime},
		"a footprint outside the repository":      {strings.Replace(plain, "Abcd-Claimed-At", "Abcd-Footprint: ../x.go\nAbcd-Claimed-At", 1), reasonFootprint},
		"a message past the cap":                  {plain + strings.Repeat("x", MaxMessageBytes), reasonMessageTooBig},
	} {
		if got, err := ReadReservationMessage(tc.text); err == nil || err.Error() != tc.why {
			t.Errorf("%s = %+v, %v; want refused as %q", name, got, err, tc.why)
		}
	}
	extra := strings.Replace(plain, "\n\n", "\n\nwhy it was released\n\n", 1) + "Abcd-Released-Kind: handed-over\n"
	got, err := ReadReservationMessage(extra)
	if err != nil {
		t.Fatalf("a message with an unknown key and a second paragraph: %v; want read, as the drain reads it", err)
	}
	if got.Lead != "abcd reservation released: iss-5\n\nwhy it was released" || !slices.Equal(got.Other, []string{"Abcd-Released-Kind: handed-over"}) {
		t.Fatalf("lead %q, other %q; want the two paragraphs and the unknown key", got.Lead, got.Other)
	}
}
