package cli

import (
	"fmt"
	"io"
	"strconv"

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

// renderBoardReviews writes the reviews heading and one line per folder,
// stalest first: a `!` on a row past the threshold, the commits since its pin,
// the pin's short sha, and what it reviewed.
func renderBoardReviews(w io.Writer, b *reviews.Board) {
	if b == nil {
		return
	}
	fmt.Fprintf(w, "  reviews:    %s, %d past %d commits since the pin on %s — re-run those marked !\n",
		countOf(len(b.Rows), "folder"), b.StaleCount(), b.Threshold, termsafe.Sanitize(b.DefaultRef))
	for _, r := range b.Rows {
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
		if r.Kind == reviews.KindReceipt {
			what = "receipt: " + r.Scope
		}
		if r.State == reviews.StateUnreachable {
			what += " (pin not in this history)"
		}
		fmt.Fprintf(w, "    %s %5s  %-8s  %s\n", flag, since, pin, termsafe.Sanitize(what))
	}
}
