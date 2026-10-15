package repolint

import (
	"path/filepath"

	"github.com/intentdriven/abcd/internal/core/decisions"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// decisionDurability checks that decisions are recorded where they survive a
// clone: the committed decisions folder, .abcd/work/decisions/, one file per
// decision (spc-2610110931381316, decision 12).
//
// A repository adopted before the folder may keep only a single DECISIONS.md.
// Its history survives a clone, but a new decision has no folder to go to, so
// that is a warning naming the install that lays one; the log itself stays as
// frozen history and is never asked to change. Decisions kept only in the
// gitignored local tier are the sharper failure the rule names specifically:
// they vanish on the next checkout.
type decisionDurability struct{}

// frozenLogRel is a managed repository's single decisions log, read here only
// to tell a repository with history from one with none. It is the same path as
// lint.DecisionsLedger, named apart because that constant belongs to the
// decisions-append gate, which leaves with abcd's own log.
const frozenLogRel = ".abcd/work/DECISIONS.md"

func (decisionDurability) Meta() RuleMeta {
	return RuleMeta{
		ID:         "decision-durability",
		Severity:   SeverityWarn,
		Fix:        "record decisions in the committed " + decisions.RelDir + "/ folder, one file each (abcd decide \"<text>\"); `abcd ahoy install` lays the folder",
		PolicyInfo: "decisions kept only in the gitignored layer do not survive a clone; the shared working tier's decisions folder is where a future session looks for them",
	}
}

func (decisionDurability) Where(Context) bool { return true }

func (decisionDurability) Eval(ctx Context) ([]Finding, error) {
	folder, err := durablyPresentDir(ctx.RepoRoot, decisions.RelDir)
	if err != nil {
		return nil, err
	}
	if folder {
		return nil, nil
	}
	finding := Finding{
		RuleID:   "decision-durability",
		Severity: SeverityWarn,
		File:     decisions.RelDir,
		Message:  "no committed " + decisions.RelDir + "/ folder; run `abcd ahoy install` to lay it",
	}

	// No durable folder. A frozen log alone is history with nowhere for the
	// next decision to go; decisions only in the gitignored layer are the
	// sharper case, and outrank it.
	local, err := fsutil.Exists(filepath.Join(ctx.RepoRoot, filepath.FromSlash(".abcd/.work.local/DECISIONS.md")))
	if err != nil {
		return nil, err
	}
	frozen, err := durablyPresent(ctx.RepoRoot, frozenLogRel)
	if err != nil {
		return nil, err
	}
	switch {
	case local:
		finding.Message = "decisions live only in the gitignored .abcd/.work.local/ layer — they will not survive a clone"
	case frozen:
		finding.Message = "only a frozen " + frozenLogRel + " is kept, and a new decision has no " + decisions.RelDir +
			"/ folder to go to; run `abcd ahoy install` to lay it (the log stays as it is)"
	}
	return []Finding{finding}, nil
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

// durablyPresentDir reports whether rel is a directory that is not gitignored.
// A pattern can name a directory with or without its trailing slash, so both
// spellings are asked. As with durablyPresent, a git that cannot answer leaves
// presence alone to decide.
func durablyPresentDir(root, rel string) (bool, error) {
	isDir, err := fsutil.IsDir(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil || !isDir {
		return false, err
	}
	if gitutil.InRepo(root) && (gitutil.IsIgnored(root, rel) || gitutil.IsIgnored(root, rel+"/")) {
		return false, nil
	}
	return true, nil
}
