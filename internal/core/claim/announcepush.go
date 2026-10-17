package claim

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

// The announcement before abcd's own pushes (iss-2610161645224876,
// dcn-2610161645360719, dcn-2610170655202547): before `abcd land` pushes a
// branch, and before a build or drain lane's landing pushes its branch, abcd
// posts "merging <item>" for each item the push is work for, through Announce
// itself, and pushes only once each has reached origin. A colleague's next
// status read therefore shows the announcement before the branch is on
// origin. A push a person types by hand with git posts nothing; no hook
// announces.

// PushKind is the kind abcd's own pushes announce.
const PushKind = "merging"

// PushItems is the announcement items among values, each in its canonical
// form, the first of each kept, in the order given: an issue or intent id or a
// pull-request number written "#N". Every other value (a spec, a decision, an
// ADR, text) is not an item and is left out, a bare number among them: values
// are a trailer's words, and the 3 of "Refs: spc-9 step 3" is not pull
// request #3. `abcd announce` still takes a bare N, since a shell reads "#N"
// as a comment; that is announcementItem, which this does not change.
func PushItems(values []string) []string {
	var out []string
	for _, v := range values {
		if !strings.HasPrefix(v, "#") && prNumberRe.MatchString(v) {
			continue
		}
		canon, _, err := announcementItem(v)
		if err != nil || slices.Contains(out, canon) {
			continue
		}
		out = append(out, canon)
	}
	return out
}

// PushNotice is what became of one item's announcement before a push: sent,
// already standing (Live), or not made because the record is not public yet
// (NotPublic), which an announcement may never name (C4).
type PushNotice struct {
	Item      string
	Branch    string
	Live      bool
	NotPublic bool
	ExpiresAt time.Time
}

// Line is the notice in one line, every value in it fixed text or a field
// held to its grammar.
func (n PushNotice) Line() string {
	switch {
	case n.NotPublic:
		return "did not announce " + n.Item + ": it is not on " + Remote + "'s default branch yet, and an announcement names only a public record"
	case n.Live:
		return PushKind + " " + n.Item + " was already announced on " + n.Branch + ", standing until " + stamp(n.ExpiresAt)
	}
	return "announced " + PushKind + " " + n.Item + " on " + n.Branch + ", standing until " + stamp(n.ExpiresAt)
}

// PushAnnounceError is an announcement a push owed that could not be
// published: the push it was owed before is not made. Item is the item, Err
// what Announce answered (a refusal, "not sent", or an *AnnounceError that
// may have reached origin).
type PushAnnounceError struct {
	Item string
	Err  error
}

func (e *PushAnnounceError) Error() string {
	msg := fmt.Sprintf("the announcement %s %s could not be published: %s", PushKind, e.Item, redacted(e.Err))
	var ae *AnnounceError
	if errors.As(e.Err, &ae) && ae.Pushes > 0 {
		msg += fmt.Sprintf("; a refused push to %s may still have reached %s", ae.Branch, Remote)
	}
	return msg
}

func (e *PushAnnounceError) Unwrap() error { return e.Err }

// AnnounceBeforePush posts "merging <item>" for each of items, in order,
// through Announce with UnlessLive, so an item already announced and still
// standing is not announced again; it is the one path abcd's pushes announce
// by. A record not yet on origin's default branch is not announced and is
// reported NotPublic, since an announcement names only a public record. Any
// other failure stops it, answering the notices made so far and a
// *PushAnnounceError: the caller then pushes nothing (dcn-2610170704189712,
// under dcn-2610170655202547). It returns only once every
// announcement it reports sent has reached origin.
func AnnounceBeforePush(root string, items []string, o Options) ([]PushNotice, error) {
	var out []PushNotice
	for _, item := range items {
		res, err := Announce(root, AnnounceRequest{Kind: PushKind, Item: item, UnlessLive: true}, o)
		var np *NotPublicError
		switch {
		case errors.As(err, &np):
			out = append(out, PushNotice{Item: np.Item, NotPublic: true})
		case err != nil:
			return out, &PushAnnounceError{Item: item, Err: err}
		default:
			out = append(out, PushNotice{Item: res.Item, Branch: res.Branch, Live: res.Live, ExpiresAt: res.ExpiresAt})
		}
	}
	return out, nil
}
