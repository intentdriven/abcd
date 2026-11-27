package landings

// spotcheck.go hands the spot-check's draw this reader (spc-2610301925429250
// approach 4). The spot-check cannot import this package, which imports
// internal/core/landing, which imports it; so the reader registers itself
// there when this package is linked, as the merge gate registers itself with
// claim for a re-arm.

import (
	"context"

	"github.com/intentdriven/abcd/internal/core/spotcheck"
)

func init() {
	spotcheck.RegisterLandingsReader(func(ctx context.Context, root, from, to string) ([]spotcheck.LandingRow, error) {
		return spotcheckRows(ctx, root, from, to, Options{})
	})
}

// spotcheckRows are the rows of from (exclusive) to to (inclusive) as the
// draw reads them: the class, the rule, the trailers and the commit, and
// whether the row's approval could not be settled because the forge's answer
// was not read. The reader's words for why it classed a row stay here.
func spotcheckRows(ctx context.Context, root, from, to string, o Options) ([]spotcheck.LandingRow, error) {
	l, err := List(ctx, root, Range{From: from, To: to}, o)
	if err != nil {
		return nil, err
	}
	rows := make([]spotcheck.LandingRow, 0, len(l.Rows))
	for _, r := range l.Rows {
		rows = append(rows, spotcheck.LandingRow{
			Commit: r.Commit, When: r.When, PR: r.PR, MergedHead: r.MergedHead, Class: r.Class,
			RuleKind: r.Rule.Kind, RuleID: r.Rule.ID, RulePath: r.Rule.Path, Lane: r.Lane, Mode: r.Mode,
			Resolves: r.Resolves, Delivers: r.Delivers, Unattended: r.Unattended,
			ForgeUnread: r.ForgeUnreadable != "" || r.Approval.Source == SourceForgeUnread,
		})
	}
	return rows, nil
}
