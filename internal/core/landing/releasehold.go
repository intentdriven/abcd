package landing

import (
	"errors"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// releaseHold is the release hold at `abcd land` (itd-2609151838312703 C7,
// spc-2610150728396006 step 4), made before land pushes, arms or queues
// anything: the land fence's own judgement (claim.FetchMergeHold), from the
// checkout land runs in at root, whose copy key and local tier are the ones
// the reservation verbs run there use, its username read under home. It
// fetches the reservations and origin's release tags, and refuses, held as
// contention, while a release another session is cutting holds every merge,
// naming each release, its holder and until when; the cutting copy's own
// landing is its ruling. A fetch that fails is refused too, since a hold land
// cannot see may be there, but as a refusal and not a hold: nobody is known
// to hold anything. A checkout with no remote named origin is not held. It
// returns the notice naming each release dated ahead that the hold did not
// count (claim.MergeHold.Notice), "" when there is none.
func releaseHold(root, home string) (string, error) {
	m, err := claim.FetchMergeHold(root, time.Now(), claim.Options{Home: home})
	var fe *claim.FetchError
	switch {
	case errors.As(err, &fe):
		return "", refuse(fsutil.RedactHome("could not read the reservations from "+claim.Remote+": "+fe.Reason+
			", so no release hold could be judged and abcd land pushes, arms and queues nothing"),
			"run abcd land again once origin answers")
	case err != nil:
		return "", refuse("the release holds could not be read: "+fsutil.RedactHome(err.Error()),
			"settle what it names, then run abcd land again; nothing was pushed")
	case m.Held():
		return "", &Refusal{
			Reason: clean(m.Reason() + ", so abcd land pushes, arms and queues nothing"),
			Fix:    clean(m.Waits() + ", then run abcd land again"),
			Held:   true,
		}
	}
	return clean(m.Notice()), nil
}

// ReleaseHoldForTest is releaseHold, for the implement loop's parity test,
// which reads it from outside this package.
func ReleaseHoldForTest(root, home string) (string, error) { return releaseHold(root, home) }
