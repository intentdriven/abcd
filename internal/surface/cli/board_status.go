package cli

import (
	"fmt"
	"io"
	"time"

	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/core/statusblock"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// statusBlockStore names the per-repository record in the checkout-root
// refusal.
const statusBlockStore = "the intent store"

// boardStatus composes the board's Now / Next / Later block
// (itd-2609212103568351) for the checkout containing cwd, from the one read the
// site's Status page renders too: the shelves, the readiness gate, and the
// build's state file through the loop. It is nil outside a checkout abcd
// manages, and when the record cannot be read (said on stderr): the board
// itself never fails on the block.
func boardStatus(cwd string, stderr io.Writer) *statusblock.Block {
	root, err := gitutil.CheckoutRoot(cwd, statusBlockStore)
	if err != nil || !ahoy.Managed(root) {
		return nil
	}
	b, err := statusblock.Read(root, loop.StatusLanes, loop.StatusPeers)
	if err != nil {
		fmt.Fprintf(stderr, "abcd: the Now / Next / Later block is omitted — %s\n", termsafe.Sanitize(fsutil.RedactHome(err.Error())))
		return nil
	}
	markReserved(root, &b, time.Now(), stderr)
	return &b
}

// markReserved names on each row the reservation another copy holds on its
// record, read from disk alone as the checkout last fetched the reservations
// (no fetch, no push); a view that cannot be read is said on stderr and marks
// nothing.
func markReserved(root string, b *statusblock.Block, now time.Time, stderr io.Writer) {
	snap, err := claim.View(root, now)
	if err != nil {
		fmt.Fprintf(stderr, "abcd: the reservations are omitted from the block — %s\n", termsafe.Sanitize(fsutil.RedactHome(err.Error())))
		return
	}
	for _, rows := range [][]statusblock.Row{b.Now, b.Next, b.Later} {
		for i := range rows {
			rows[i].Reserved = termsafe.Sanitize(snap.HeldBy(rows[i].ID, now))
		}
	}
}
