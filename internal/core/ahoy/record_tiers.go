package ahoy

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// ---------------------------------------------------------------------------
// the committed tiers
// ---------------------------------------------------------------------------

// The committed tiers of the three-tier layout (.abcd/development/ and
// .abcd/work/) are what the repository lint's three-tier-layout rule requires,
// so an install that reports a repository set up creates them
// (iss-2610071538028804). The list is the lint package's own
// (lint.CommittedTiers): the rule and the install read one list, never two.
//
// git keeps no empty directory, so the shared tier is seeded with its decision
// log (lint.DecisionsLedger): the tier then survives a clone, and the lint's
// decision-durability rule finds the log it asks for. The seed is a header
// that says how the log is kept and records no decision.

// decisionsLedgerGapID is the gap install closes by seeding the decision log
// into a shared tier that exists without one. A missing shared tier is its own
// gap, whose fix seeds the log too.
const decisionsLedgerGapID = "skeleton.decisions_ledger_missing"

// committedTierGapID is the gap install closes by creating tier.
func committedTierGapID(t lint.CommittedTier) string {
	return "skeleton." + path.Base(t.Rel) + "_tier_missing"
}

// decisionsLedgerSeed is the decision log's starter text: how the log is kept,
// and no entry.
const decisionsLedgerSeed = `# DECISIONS

Append-only, one line per decision, newest last, each line starting with its
date (YYYY-MM-DD). Correct an earlier entry by adding a new dated line, never
by editing the old one. Architecture-shaping decisions graduate to an ADR under
../development/decisions/adrs/ (abcd decide).
`

// detectCommittedTiers reports each committed tier that is not a directory,
// judged as the lint judges it (a symlink to a directory satisfies both), and a
// shared tier that holds no decision log.
func detectCommittedTiers(cwd string) []Gap {
	var gaps []Gap
	for _, t := range lint.CommittedTiers() {
		isDir, _ := fsutil.IsDir(filepath.Join(cwd, filepath.FromSlash(t.Rel)))
		if isDir {
			continue
		}
		detail := "The " + t.Label + " is absent (or is not a directory). It is a committed part of abcd's layout, and abcd lint reports a repository without it as an error."
		if t.Rel == path.Dir(lint.DecisionsLedger) {
			detail += " Install seeds it with an empty " + path.Base(lint.DecisionsLedger) + " so it survives a clone."
		}
		gaps = append(gaps, Gap{
			ID: committedTierGapID(t), Category: SafeAutocreate, Scope: "repo",
			Title:    t.Rel + "/ missing",
			Detail:   detail,
			FixHint:  "ahoy install creates it as a real directory.",
			Required: true, Resolvable: true,
		})
	}
	ledgerDir := filepath.Join(cwd, filepath.FromSlash(path.Dir(lint.DecisionsLedger)))
	if isDir, _ := fsutil.IsDir(ledgerDir); isDir {
		if _, err := os.Lstat(filepath.Join(cwd, filepath.FromSlash(lint.DecisionsLedger))); errors.Is(err, fs.ErrNotExist) {
			gaps = append(gaps, Gap{
				ID: decisionsLedgerGapID, Category: SafeAutocreate, Scope: "repo",
				Title:    lint.DecisionsLedger + " missing",
				Detail:   "The shared tier holds no decision log. Decisions recorded there are committed and survive a clone; abcd lint warns without one.",
				FixHint:  "ahoy install seeds it with a header that records no decision.",
				Required: true, Resolvable: true,
			})
		}
	}
	return gaps
}

// stepCommittedTiers creates each missing committed tier as a real directory,
// proving every level real (fsutil.EnsureRealDirAll) from the checkout root
// resolved through its symlinks, as stepLocalTier does, and then seeds the
// decision log. The log is created exclusively: one that appeared since
// detection, or one already there, is kept as it is.
func (a *applyCtx) stepCommittedTiers() {
	if !a.approved[SafeAutocreate] {
		return
	}
	root := fsutil.RealExistingPath(a.cwd)
	ledgerTier := path.Dir(lint.DecisionsLedger)
	seed := a.has(decisionsLedgerGapID)
	for _, t := range lint.CommittedTiers() {
		if !a.has(committedTierGapID(t)) {
			continue
		}
		if err := fsutil.EnsureRealDirAll(root, t.Rel, 0o755); err != nil {
			a.refuse("refused to create " + t.Rel + "/: " + tierRefusalReason(root, err) +
				". abcd never replaces what stands at a tier's path; remove it and re-run `abcd ahoy install`.")
			if t.Rel == ledgerTier {
				seed = false
			}
			continue
		}
		a.note(writeRecordTiers, filepath.Join(a.cwd, filepath.FromSlash(t.Rel)))
		if t.Rel == ledgerTier {
			seed = true
		}
	}
	if !seed {
		return
	}
	wrote, err := createRepoFile(a.cwd, lint.DecisionsLedger, []byte(decisionsLedgerSeed))
	if wrote {
		a.note(writeDecisionsLedger, filepath.Join(a.cwd, filepath.FromSlash(lint.DecisionsLedger)))
	}
	if err != nil {
		a.refuse("could not write " + lint.DecisionsLedger + ": " + errText(err))
	}
}

// tierRefusalReason renders why a tier could not be created: the level that is
// not a real directory, repository-relative, or the error itself.
func tierRefusalReason(root string, err error) string {
	var pe *os.PathError
	if errors.Is(err, fsutil.ErrNotRealDir) && errors.As(err, &pe) {
		level := "the checkout root"
		if rel, relErr := filepath.Rel(root, pe.Path); relErr == nil && rel != "." && !strings.HasPrefix(rel, "..") {
			level = filepath.ToSlash(rel)
		}
		return level + " is not a real directory (a symlink, or a file, stands there)"
	}
	return errText(err)
}
