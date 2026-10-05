package cli

import (
	"fmt"
	"io"

	"github.com/intentdriven/abcd/internal/core/ahoy"
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
	return &b
}
