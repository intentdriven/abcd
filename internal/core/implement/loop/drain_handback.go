package loop

// drain_handback.go is the drain's side of the hand-back published on origin
// (spc-2610161044461448 step 3; iss-2610161015370582; dcn-2610161040365515).
// The tombstone itself is claim's: the release of a handed-back issue's
// reservation names the hand-back's kind and the blob id of the issue's record
// on origin's default branch (releaseReservation), and a drain's reservation
// backs off it while that record is unchanged. What is here is the promotion
// a move planned from origin may not make: `capture promote` edits this
// checkout's record, so a checkout that does not hold the record as origin
// does promotes nothing, and the hand-back is published alone. That a lane of
// any issue run publishes its hand-back, that a move passes a standing one
// over from its view before any push, and how a checkout is judged to hold a
// record as origin does, are dcn-2610161412284018.

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/peers"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// notHeldAsOrigin says why this checkout does not hold issue's open record as
// origin's default branch does at the commit the move planned from: absent
// here, held here as another record (its blob, as git would store this
// checkout's file, is not origin's), or not open on origin. "" when it holds
// it as origin does, and with no origin (nil), where the move planned from
// the checkout's own ledger.
func notHeldAsOrigin(repoRoot, issue string, origin *DrainOrigin) (string, error) {
	if origin == nil {
		return "", nil
	}
	recs, err := peers.RecordsAt(repoRoot, origin.Commit)
	if err != nil {
		return "", refuse(StageDrain, "", "", "could not read "+issue+"'s record on "+origin.Ref+" to route its hand-back: "+fsutil.RedactHome(err.Error()),
			"settle what it names, then run `abcd drain` again; the hand-back is routed then")
	}
	want := ""
	if rs := recs[issue]; len(rs) == 1 && rs[0].Folder == string(capture.StateOpen) {
		want = rs[0].Blob
	}
	if want == "" {
		return fmt.Sprintf("%s holds no one open record of %s", origin.Ref, issue), nil
	}
	if !validIssueKey(issue) {
		return "this checkout holds no record of " + issue, nil
	}
	open := filepath.Join(repoRoot, filepath.FromSlash(recordid.IssuesRelDir), string(capture.StateOpen))
	matches, err := filepath.Glob(filepath.Join(open, issue+"-*.md"))
	if err != nil || len(matches) == 0 {
		return "this checkout holds no record of " + issue, nil
	}
	if len(matches) > 1 {
		return "this checkout holds " + issue + " in more than one record", nil
	}
	rel := recordid.IssuesRelDir + "/" + string(capture.StateOpen) + "/" + filepath.Base(matches[0])
	got, err := gitutil.Run(repoRoot, "hash-object", "--", rel)
	if err != nil {
		return "this checkout's record of " + issue + " could not be read", nil
	}
	if strings.TrimSpace(got) != want {
		return "this checkout's record of " + issue + " is not the one " + claim.Remote + "'s default branch holds", nil
	}
	return "", nil
}
