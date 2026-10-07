package repolint

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// threeTierLayout checks the committed three-tier .abcd/ layout: the durable
// record (.abcd/development/) and the shared working tier (.abcd/work/) must be
// present, the local-ephemeral tier (.abcd/.work.local/), when present, must
// be gitignored so per-worktree state never leaks into history, and the
// local tier's conventional artefacts (NEXT.md, scratch/, logs/) must not sit
// directly in a committed tier — a handover file carrying machine-local detail
// in .abcd/work/ is committed, pushed, and public before anything flags it.
//
// Presence of .work.local is NOT required — it is created on demand and a fresh
// clone has none. Requiring it would flag every clean checkout. The load-bearing
// assertion is "if present, ignored", which is what stops decisions and logs
// leaking. (Divergence from the plan's literal "present and gitignored",
// recorded in DECISIONS.md.)
type threeTierLayout struct{}

func (threeTierLayout) Meta() RuleMeta {
	return RuleMeta{
		ID:         "three-tier-layout",
		Severity:   SeverityError,
		Fix:        "create the missing .abcd/ tier; ensure .abcd/.work.local/ is listed in .gitignore; move local-tier artefacts (NEXT.md, scratch/, logs/) out of the committed tiers into .abcd/.work.local/",
		PolicyInfo: "the three-tier layout separates the durable record, shared working state, and per-worktree ephemera; the local tier must be gitignored so it never merge-conflicts or leaks",
	}
}

func (threeTierLayout) Where(Context) bool { return true }

func (threeTierLayout) Eval(ctx Context) ([]Finding, error) {
	var out []Finding

	// The committed tiers are read from the one list `abcd ahoy install`
	// creates them from, so an install that reports the repository set up
	// leaves nothing here to find (iss-2610071538028804).
	for _, t := range lint.CommittedTiers() {
		tier := struct{ rel, label string }{t.Rel, t.Label}
		// A tier is a directory: a regular file at the tier path does not satisfy
		// the convention, so check the type, not mere presence. (IsDir follows a
		// symlink, so a symlink-to-directory does satisfy it — acceptable for a
		// layout check, which is not the owned-store trust boundary.)
		isDir, err := fsutil.IsDir(filepath.Join(ctx.RepoRoot, filepath.FromSlash(tier.rel)))
		if err != nil {
			return nil, err
		}
		if !isDir {
			out = append(out, Finding{
				RuleID:   "three-tier-layout",
				Severity: SeverityError,
				File:     tier.rel,
				Message:  "missing the " + tier.label + " (must be a directory)",
			})
			continue
		}

		// Local-tier artefacts in a committed tier: NEXT.md, scratch/ and logs/
		// are the local-ephemeral tier's conventional contents, so their presence
		// in a committed tier is a placement error of the leak class — per-worktree
		// ephemera about to enter (or already in) history. Presence is checked on
		// the filesystem, like the tiers themselves: an untracked NEXT.md in
		// .abcd/work/ is one `git add -A` from being committed. The listing is
		// no-follow: the NAME occupying the path is the violation regardless of
		// what it is — a dangling symlink named NEXT.md still gets committed, and
		// its target string can itself be a private path.
		found, err := misplacedLocalArtefacts(ctx.RepoRoot, tier.rel, tier.label)
		if err != nil {
			return nil, err
		}
		out = append(out, found...)
	}

	// The .abcd/ root is one directory off the modelled incident and just as
	// committed: a handover dropped there rides the same `git add -A`.
	rootFound, err := misplacedAt(ctx.RepoRoot, ".abcd", ".abcd/ root", localArtefactNames)
	if err != nil {
		return nil, err
	}
	out = append(out, rootFound...)

	// The local tier: only an issue when it is present but not gitignored, and
	// only when git can actually answer — git-absent is "cannot tell", never a
	// silent pass claiming it is ignored.
	localRel := ".abcd/.work.local"
	localPresent, err := fsutil.Exists(filepath.Join(ctx.RepoRoot, filepath.FromSlash(localRel)))
	if err != nil {
		return nil, err
	}
	if localPresent && gitutil.InRepo(ctx.RepoRoot) {
		if !gitutil.IsIgnored(ctx.RepoRoot, localRel+"/") {
			out = append(out, Finding{
				RuleID:   "three-tier-layout",
				Severity: SeverityError,
				File:     localRel,
				Message:  "the local-ephemeral tier .abcd/.work.local/ is present but not gitignored — its contents would be committed",
			})
		}
	}

	return out, nil
}

// localArtefactNames are the local-ephemeral tier's conventional contents. They
// are matched in any case: on a case-sensitive filesystem `next.md` is a
// different file from NEXT.md and is committed just the same (iss-173).
var localArtefactNames = []string{"NEXT.md", "scratch", "logs"}

// handoverName is the one artefact flagged at ANY depth in a committed tier. The
// handover file is a name the local tier owns outright, so a NEXT.md nested
// under a tier is the incident one directory down. scratch/ and logs/ are
// ordinary words a durable record may legitimately nest (a study's own logs/),
// so they are flagged only where the local tier would put them: directly under
// a tier root and at the .abcd/ root.
const handoverName = "NEXT.md"

// misplacedLocalArtefacts reports the local-tier artefacts in the committed tier
// at tierRel: any of the three names directly under it, and a handover file at
// any depth below that. The walk never follows a symlink, so a linked directory
// is judged by its own name and never walked into.
func misplacedLocalArtefacts(repoRoot, tierRel, label string) ([]Finding, error) {
	out, err := misplacedAt(repoRoot, tierRel, label, localArtefactNames)
	if err != nil {
		return nil, err
	}
	tierAbs := filepath.Join(repoRoot, filepath.FromSlash(tierRel))
	err = filepath.WalkDir(tierAbs, func(p string, d fs.DirEntry, werr error) error {
		if werr != nil {
			return werr
		}
		// Direct children were judged by misplacedAt; a handover below them is
		// what the walk is for.
		if filepath.Dir(p) == tierAbs || p == tierAbs {
			return nil
		}
		if !strings.EqualFold(d.Name(), handoverName) {
			return nil
		}
		rel, rerr := filepath.Rel(repoRoot, p)
		if rerr != nil {
			return rerr
		}
		out = append(out, artefactFinding(filepath.ToSlash(rel), d.Name(), label))
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// misplacedAt reports each entry directly inside dirRel whose name is one of
// names in any case. An absent directory holds nothing to report.
func misplacedAt(repoRoot, dirRel, label string, names []string) ([]Finding, error) {
	entries, err := os.ReadDir(filepath.Join(repoRoot, filepath.FromSlash(dirRel)))
	if os.IsNotExist(err) || errors.Is(err, syscall.ENOTDIR) {
		// Absent, or not a directory at all: nothing sits inside it, and the
		// missing tier is reported as itself.
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Finding
	for _, e := range entries {
		for _, n := range names {
			if strings.EqualFold(e.Name(), n) {
				out = append(out, artefactFinding(dirRel+"/"+e.Name(), e.Name(), label))
				break
			}
		}
	}
	return out, nil
}

// artefactFinding is the one finding shape every misplacement produces, naming
// the path as it is spelled on disk.
func artefactFinding(rel, name, label string) Finding {
	return Finding{
		RuleID:   "three-tier-layout",
		Severity: SeverityError,
		File:     rel,
		Message:  "local-tier artefact " + name + " found in the " + label + " — per-worktree ephemera must never enter a committed tier",
		Fix:      "move " + rel + " to the local-ephemeral tier .abcd/.work.local/",
	}
}
