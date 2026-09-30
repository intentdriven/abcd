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
	b, err := statusblock.Read(root, loop.StatusLanes)
	if err != nil {
		fmt.Fprintf(stderr, "abcd: the Now / Next / Later block is omitted — %s\n", termsafe.Sanitize(fsutil.RedactHome(err.Error())))
		return nil
	}
	return &b
}

// renderBoardStatus writes the block: a heading with the three counts, then
// Now and Next, one row per intent (Next in the pick order) — its id, its
// title, and in brackets what places it there (its lane state or "next up")
// and the release it targets —
// then Later as a count of intents alone (ruling BV1 of 2026-09-29): its rows,
// with the gating checks each fails, are in --json and on the site's Status
// page.
func renderBoardStatus(w io.Writer, b *statusblock.Block) {
	if b == nil {
		return
	}
	fmt.Fprintf(w, "  status:     Now %d · Next %d · Later %d\n", len(b.Now), len(b.Next), len(b.Later))
	for _, list := range []struct {
		name string
		rows []statusblock.Row
	}{{"Now", b.Now}, {"Next", b.Next}} {
		fmt.Fprintf(w, "    %s:\n", list.name)
		if len(list.rows) == 0 {
			fmt.Fprintln(w, "      (none)")
			continue
		}
		for _, r := range list.rows {
			line := "      " + termsafe.Sanitize(r.ID) + "  " + termsafe.Sanitize(r.Title)
			if tag := statusRowTag(r); tag != "" {
				line += "  [" + tag + "]"
			}
			fmt.Fprintln(w, line)
		}
	}
	noun := "intents"
	if len(b.Later) == 1 {
		noun = "intent"
	}
	fmt.Fprintf(w, "    Later: %d %s\n", len(b.Later), noun)
}

// statusRowTag is what places a Now or Next row where it is, in words — its
// lane state or "next up"; a READY intent in no lane carries none — then the
// release the intent targets, when it names one (itd-2609212103572513
// criterion 4).
func statusRowTag(r statusblock.Row) string {
	place := statusRowPlace(r)
	if r.Target == "" {
		return place
	}
	target := "target " + termsafe.Sanitize(r.Target)
	if place == "" {
		return target
	}
	return place + "; " + target
}

// statusRowPlace is what places a Now or Next row where it is: its lane state
// or "next up", or nothing.
func statusRowPlace(r statusblock.Row) string {
	switch {
	case r.Lane != nil:
		l := r.Lane
		tag := termsafe.Sanitize(l.Step)
		if l.Lane != "" {
			tag = termsafe.Sanitize(l.Lane) + ": " + tag
		}
		if l.Awaiting != "" {
			tag += ", awaiting the " + termsafe.Sanitize(l.Awaiting)
		}
		return tag + " (" + termsafe.Sanitize(l.Run) + ")"
	case r.NextUp:
		return "next up"
	}
	return ""
}
