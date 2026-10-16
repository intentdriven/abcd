package ahoy

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/core/decisions"
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
// git keeps no empty directory, so each tier is seeded with one file, or a
// fresh clone of the installed repository loses it and fails the same rule.
// The shared tier's seed is the README of its decisions folder
// (decisions.RelDir), which the lint's decision-durability rule also asks for;
// the durable-record tier's is a README of its own. Each seed is a header that
// says what its folder is for and claims nothing about the repository.
//
// No single decisions log is seeded, and one a repository already keeps is never
// opened: it stays as that repository's frozen history (spc-2610110931381316,
// decision 7). Nothing here writes a merge attribute either: recording a
// decision adds a file of its own, so git has nothing to reconcile.

// decisionsFolderGapID is the gap install closes by seeding the decisions
// folder into a shared tier that exists without one. It replaced
// skeleton.decisions_ledger_missing, which named a file no longer seeded, with
// no alias (spc-2610110931381316, decision 11).
const decisionsFolderGapID = "skeleton.decisions_folder_missing"

// developmentReadmeGapID is the gap install closes by seeding the README into
// a durable-record tier that exists without one.
const developmentReadmeGapID = "skeleton.development_readme_missing"

// committedTierGapID is the gap install closes by creating tier.
func committedTierGapID(t lint.CommittedTier) string {
	return "skeleton." + path.Base(t.Rel) + "_tier_missing"
}

// decisionsFolderSeed is the decisions folder's README: what the folder holds
// and the command that records and lists it, and no decision. It is static:
// recording a decision adds a file beside it and never edits it.
const decisionsFolderSeed = `# Decisions

One file per decision, dcn-<id>-<slug>.md, each opening with its date. Record
one with abcd decide "<text>", which adds a new file and edits nothing else, so
two branches that each record a decision merge cleanly. List them, newest
first, with abcd decide. Correct an earlier decision by recording a new one,
never by editing the old file. Architecture-shaping decisions graduate to an
ADR under ../../development/decisions/adrs/ (abcd decide --adr).
`

// developmentReadmeSeed is the durable-record tier's starter text: what the
// tier holds, and nothing else.
const developmentReadmeSeed = `# Development record

The durable record of this repository: the intents, decisions, plans and
research the work is built from, kept in the repository so that every clone
carries it.
`

// tierSeed is the one file a committed tier is seeded with.
type tierSeed struct {
	tier    string // the tier's repo-relative path, as lint.CommittedTiers names it
	file    string // the seed's repo-relative path
	present string // the path whose presence means the tier needs no seed
	gapID   string // the gap raised when the tier exists without present
	content string
	kind    writeKind
	detail  string // the gap's detail
}

// tierSeeds is the seed of each committed tier. TestEveryCommittedTierHasASeed
// holds it to lint.CommittedTiers, so a tier added there without a seed fails.
func tierSeeds() []tierSeed {
	return []tierSeed{
		{
			tier: ".abcd/development", file: ".abcd/development/README.md", present: ".abcd/development/README.md",
			gapID: developmentReadmeGapID, content: developmentReadmeSeed, kind: writeDevelopmentReadme,
			detail: "The durable-record tier holds no README. git keeps no empty folder, so without a file in it the tier is missing from every clone, and abcd lint reports that as an error.",
		},
		{
			// The folder, not its README, is what closes the gap: a folder that
			// already holds decisions needs no seed to survive a clone.
			tier: path.Dir(decisions.RelDir), file: decisions.RelDir + "/README.md", present: decisions.RelDir,
			gapID: decisionsFolderGapID, content: decisionsFolderSeed, kind: writeDecisionsFolder,
			detail: "The shared tier holds no decisions folder (" + decisions.RelDir + "/). A decision recorded there is a file of its own, committed, so it survives a clone and merges without conflict; abcd lint warns without one.",
		},
	}
}

// seedFor returns tier's seed, if it has one.
func seedFor(tier string) (tierSeed, bool) {
	for _, s := range tierSeeds() {
		if s.tier == tier {
			return s, true
		}
	}
	return tierSeed{}, false
}

// detectCommittedTiers reports each committed tier that is not a directory,
// judged as the lint judges it (a symlink to a directory satisfies both), and
// each tier that exists without its seed file. A missing tier raises only its
// own gap: its fix seeds the file too.
func detectCommittedTiers(cwd string) []Gap {
	var gaps []Gap
	for _, t := range lint.CommittedTiers() {
		seed, seeded := seedFor(t.Rel)
		isDir, _ := fsutil.IsDir(filepath.Join(cwd, filepath.FromSlash(t.Rel)))
		if !isDir {
			detail := "The " + t.Label + " is absent (or is not a directory). It is a committed part of abcd's layout, and abcd lint reports a repository without it as an error."
			if seeded {
				detail += " Install seeds it with " + path.Base(seed.file) + " so it survives a clone."
			}
			gaps = append(gaps, Gap{
				ID: committedTierGapID(t), Category: SafeAutocreate, Scope: "repo",
				Title:    t.Rel + "/ missing",
				Detail:   detail,
				FixHint:  "ahoy install creates it as a real directory.",
				Required: true, Resolvable: true,
			})
			continue
		}
		if !seeded {
			continue
		}
		if _, err := os.Lstat(filepath.Join(cwd, filepath.FromSlash(seed.present))); errors.Is(err, fs.ErrNotExist) {
			gaps = append(gaps, Gap{
				ID: seed.gapID, Category: SafeAutocreate, Scope: "repo",
				Title:    seed.present + " missing",
				Detail:   seed.detail,
				FixHint:  "ahoy install seeds it with a short header that claims nothing about the repository.",
				Required: true, Resolvable: true,
			})
		}
	}
	return gaps
}

// stepCommittedTiers creates each missing committed tier as a real directory,
// proving every level real (fsutil.EnsureRealDirAll) from the checkout root
// resolved through its symlinks, as stepLocalTier does, and then seeds each
// tier it created, or found without its seed. A seed is created exclusively:
// one that appeared since detection, or one already there, is kept as it is.
func (a *applyCtx) stepCommittedTiers() {
	if !a.approved[SafeAutocreate] {
		return
	}
	root := fsutil.RealExistingPath(a.cwd)
	for _, t := range lint.CommittedTiers() {
		seed, seeded := seedFor(t.Rel)
		plant := seeded && a.has(seed.gapID)
		if a.has(committedTierGapID(t)) {
			if err := fsutil.EnsureRealDirAll(root, t.Rel, 0o755); err != nil {
				a.refuse("refused to create " + t.Rel + "/: " + tierRefusalReason(root, err) +
					". abcd never replaces what stands at a tier's path; remove it and re-run `abcd ahoy install`.")
				continue
			}
			a.note(writeRecordTiers, filepath.Join(a.cwd, filepath.FromSlash(t.Rel)))
			plant = seeded
		}
		if !plant {
			continue
		}
		wrote, err := createRepoFile(a.cwd, seed.file, []byte(seed.content))
		if wrote {
			a.note(seed.kind, filepath.Join(a.cwd, filepath.FromSlash(seed.file)))
		}
		if err != nil {
			a.refuse("could not write " + seed.file + ": " + errText(err))
		}
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
