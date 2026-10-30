package claim

import (
	"time"

	"github.com/intentdriven/abcd/internal/localtier"
)

// LastFetched is when abcd last fetched the reservations and origin's default
// branch in the checkout at root: the age of the view a reader plans from
// without fetching (the drain's dry run, spc-2610161044461448 step 1). Zero
// when nothing here stamped a fetch, and when the stamp is one git tracks,
// which came with a commit and says nothing of this checkout's fetches.
func LastFetched(root string) time.Time {
	at := lastFetched(root, reservations)
	if at.IsZero() || localtier.Untracked(root, reservations.stampRel()) != nil {
		return time.Time{}
	}
	return at
}
