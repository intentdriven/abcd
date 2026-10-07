package repolint

import (
	"path/filepath"

	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// decisionDurability checks that architectural decisions are recorded where they
// survive a clone: a committed .abcd/work/DECISIONS.md. A DECISIONS.md that
// exists only in the gitignored local tier is the failure mode the rule names
// specifically — those decisions vanish on the next checkout.
type decisionDurability struct{}

func (decisionDurability) Meta() RuleMeta {
	return RuleMeta{
		ID:         "decision-durability",
		Severity:   SeverityWarn,
		Fix:        "record decisions in a committed .abcd/work/DECISIONS.md, not only in the gitignored .work.local/ layer",
		PolicyInfo: "decisions kept only in the gitignored layer do not survive a clone; the shared working tier is where a future session looks for them",
	}
}

func (decisionDurability) Where(Context) bool { return true }

func (decisionDurability) Eval(ctx Context) ([]Finding, error) {
	committedRel := lint.DecisionsLedger
	committed, err := durablyPresent(ctx.RepoRoot, committedRel)
	if err != nil {
		return nil, err
	}
	if committed {
		return nil, nil
	}

	// No durable DECISIONS.md. Distinguish "none at all" from the sharper
	// "decisions are only in the gitignored layer".
	local, err := fsutil.Exists(filepath.Join(ctx.RepoRoot, filepath.FromSlash(".abcd/.work.local/DECISIONS.md")))
	if err != nil {
		return nil, err
	}
	msg := "no committed .abcd/work/DECISIONS.md"
	if local {
		msg = "decisions live only in the gitignored .abcd/.work.local/ layer — they will not survive a clone"
	}
	return []Finding{{
		RuleID:   "decision-durability",
		Severity: SeverityWarn,
		File:     committedRel,
		Message:  msg,
	}}, nil
}

// durablyPresent reports whether rel exists and is committed (present and not
// gitignored). When git cannot answer, presence alone is accepted — cannot-tell
// must not manufacture a warning about a file that is on disk.
func durablyPresent(root, rel string) (bool, error) {
	present, err := fsutil.Exists(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil || !present {
		return false, err
	}
	if gitutil.InRepo(root) && gitutil.IsIgnored(root, rel) {
		return false, nil
	}
	return true, nil
}
