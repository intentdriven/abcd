package lint

import (
	"path/filepath"

	"github.com/intentdriven/abcd/internal/localtier"
)

// ruleLocalTierUntracked keeps the local tier out of the history. The tier is
// each checkout's own (its run and drain state, receipts, holds and scratch),
// and a file there that a commit carries reaches every clone, where abcd's
// readers refuse it as this checkout's state (localtier.Untracked). That
// refusal is the guard; this rule moves it to the change that tracks the file,
// before any clone checks it out.
const ruleLocalTierUntracked = "local_tier_untracked"

// checkLocalTierUntracked reports every file git tracks that reaches the local
// tier, judged by the file it names, so a spelling the filesystem folds onto
// the tier, or a link or submodule standing in for it, is reported as surely
// as the tier's own name (localtier.Carried). Outside a repository nothing is
// tracked; a repository git cannot list for is an error, never a pass.
func checkLocalTierUntracked(repoRoot string, cfg RuleConfig) ([]Finding, error) {
	tracked, err := localtier.Carried(repoRoot)
	if err != nil {
		return nil, &configError{ruleLocalTierUntracked + ": asking git what it tracks under " + localtier.RelDir + ": " + err.Error()}
	}
	var out []Finding
	for _, rel := range tracked {
		out = append(out, Finding{
			File: filepath.FromSlash(rel), Line: 1, RuleID: ruleLocalTierUntracked, Severity: cfg.Severity,
			Message: "git tracks this file, but " + localtier.RelDir + " is each checkout's own and never committed: " +
				"every clone would check it out, and abcd refuses to read it there. Untrack it (git rm --cached " + rel + ")",
		})
	}
	return out, nil
}
