package cli

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/reviews"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// reviewsStore names the per-repository tree in the checkout-root refusal.
const reviewsStore = "the reviews tree"

// boardReviews composes the board's reviews member (itd-28): one row per
// folder under .abcd/work/reviews/, with the commits the default branch has
// moved since its pin. It is nil outside a checkout, when the tree holds no
// folder, and when the tree cannot be read (said on stderr): the board itself
// never fails on the reviews tree.
func boardReviews(cwd string, stderr io.Writer) *reviews.Board {
	root, err := gitutil.CheckoutRoot(cwd, reviewsStore)
	if err != nil {
		return nil
	}
	b, err := reviews.Staleness(root)
	if err != nil {
		fmt.Fprintf(stderr, "abcd: the reviews lines are omitted — %s\n", termsafe.Sanitize(fsutil.RedactHome(err.Error())))
		return nil
	}
	if len(b.Rows) == 0 {
		return nil
	}
	return &b
}

// renderBoardReviews writes the reviews heading, one line per dated review,
// stalest first — a `!` on a row past the threshold, the commits since its
// pin, the pin's short sha, and the folder — and one line for the release
// receipts. A receipt gates the release it names and is never re-run, and
// RD002 keeps every one, so each release adds a receipt that stays past the
// threshold for good: listed a row apiece they would grow the bare board by a
// flagged row per release, each asking for a re-run nobody makes. The line
// says how many there are, how far behind the oldest release it gated is, and
// how many pins this history no longer holds; --json carries every receipt as
// a row.
func renderBoardReviews(w io.Writer, b *reviews.Board) {
	if b == nil {
		return
	}
	var dated []reviews.Row
	receipts, unreachable, oldest := 0, 0, -1
	for _, r := range b.Rows {
		if r.Kind != reviews.KindReceipt {
			dated = append(dated, r)
			continue
		}
		receipts++
		switch {
		case r.CommitsSince == nil:
			unreachable++
		case *r.CommitsSince > oldest:
			oldest = *r.CommitsSince
		}
	}
	stale := 0
	for _, r := range dated {
		if r.Stale {
			stale++
		}
	}
	ref := termsafe.Sanitize(b.DefaultRef)
	heading := fmt.Sprintf("  reviews:    %s, %d past %d commits since the pin on %s",
		countOf(len(dated), "review folder"), stale, b.Threshold, ref)
	if stale > 0 {
		heading += " — re-run those marked !"
	}
	fmt.Fprintln(w, heading)
	for _, r := range dated {
		flag, since, pin := " ", "—", "unpinned"
		if r.Stale {
			flag = "!"
		}
		if r.CommitsSince != nil {
			since = strconv.Itoa(*r.CommitsSince)
		}
		if r.ReviewOfCommit != "" {
			pin = r.ReviewOfCommit[:7]
		}
		what := r.Folder
		if r.State == reviews.StateUnreachable {
			what += " (pin not in this history)"
		}
		fmt.Fprintf(w, "    %s %5s  %-8s  %s\n", flag, since, pin, termsafe.Sanitize(what))
	}
	if receipts == 0 {
		return
	}
	line := "    receipts: " + countOf(receipts, "release receipt")
	var facts []string
	if oldest >= 0 {
		which := "the oldest release gated is"
		if receipts == 1 {
			which = "the release it gated is"
		}
		facts = append(facts, fmt.Sprintf("%s %d commits behind %s", which, oldest, ref))
	}
	if unreachable > 0 {
		facts = append(facts, fmt.Sprintf("%d with a pin not in this history", unreachable))
	}
	if len(facts) > 0 {
		line += " — " + strings.Join(facts, ", ")
	}
	fmt.Fprintln(w, line+"; a receipt is not re-run, and --json lists each")
}
