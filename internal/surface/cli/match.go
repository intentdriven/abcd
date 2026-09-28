package cli

import (
	"fmt"
	"io"

	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/record/match"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// resolveMatch reads match.threshold and match.fields for a filing through the
// layered configuration reader (itd-2609212137116617). A configuration the
// reader refuses never refuses the filing, because a finding lost to a
// misspelt key is the failure the match exists to prevent: it comes back as
// the reason the match compared nothing, which the verb prints, and the record
// is filed unlinked.
func resolveMatch(stderr io.Writer, verb, repoRoot string) (*match.Config, *match.Outcome) {
	roots, notes := layered.RootsFor(repoRoot)
	for _, n := range notes {
		fmt.Fprintf(stderr, "abcd %s\n", termsafe.Sanitize(fsutil.RedactHome(n)))
	}
	cfg, err := match.LoadConfig(roots)
	if err != nil {
		o := match.Skip(match.DefaultThreshold, "the match configuration is refused ("+
			fsutil.RedactHome(err.Error())+"), so the record is filed without matching; fix or remove the key")
		return nil, &o
	}
	return &cfg, nil
}

// renderMatch prints a filing's match: each link written, each match past the
// link cap, the count of near misses (--json lists them with their scores), or
// why nothing was compared. It prints nothing when no match was asked for.
func renderMatch(w io.Writer, o *match.Outcome) {
	if o == nil {
		return
	}
	if o.Skipped != "" {
		fmt.Fprintf(w, "  not matched: %s\n", termsafe.Sanitize(o.Skipped))
	}
	for _, m := range o.Matches {
		state := "link written"
		if !m.Linked {
			state = "listed, not linked"
		}
		fmt.Fprintf(w, "  matched %s — %s (score %.3f, threshold %.2f): %s; confirm it by leaving the link, or remove the line\n",
			termsafe.Sanitize(m.ID), m.Relation, m.Score, o.Threshold, state)
	}
	if o.Skipped == "" && len(o.Matches) == 0 {
		fmt.Fprintf(w, "  no match at threshold %.2f among %d record(s)", o.Threshold, o.Compared)
		if n := len(o.NearMisses); n > 0 {
			fmt.Fprintf(w, "; %d near miss(es) below it, listed with their scores by --json", n)
		}
		fmt.Fprintln(w)
	}
}
