package claim

import "time"

// LastFetched is when abcd last fetched the reservations and origin's default
// branch in the checkout at root: the age of the view a reader plans from
// without fetching (the drain's dry run, spc-2610161044461448 step 1). Zero
// when nothing here stamped a fetch.
func LastFetched(root string) time.Time { return lastFetched(root, reservations) }
