---
id: itd-10
slug: purge-uninstall
spec_id: null
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-13]
severity: minor
---

# Full Removal When You Mean It

## Press Release

> **abcd supports `/abcd:ahoy destroy`** — a deeper uninstall that goes beyond removing the marker block. When you decide abcd isn't right for a project, `destroy` cleanly removes the marker block, the `/usr/local/bin/abcd` symlink, AND the entire `.abcd/` namespace (including lifeboat, dev artefacts, logbook, and config). The `gitignore` allowlist entries get reverted. abcd offers a final summary of what's about to be deleted with hard confirmation, then walks away. The danger is encoded in the verb itself — same convention as `rm -rf`, `git push --force`, `terraform destroy`.
>
> "Marker-only `uninstall` was great for 'I'll be back' moments," said Jack, consultant. "But sometimes a project leaves abcd entirely. Now I can `/abcd:ahoy destroy` and clean up properly without `rm -rf`'ing manually and missing things."

## Why This Matters

abcd's `uninstall` is intentionally lightweight — `/abcd:ahoy uninstall` removes only the BEGIN/END marker block from CLAUDE.md/AGENTS.md and the symlink. Everything in `.abcd/` stays put. This is correct for the common "ahoy then change my mind" case: re-running ahoy resurrects markers from existing config without re-asking.

But for projects that genuinely leave abcd (acquired, deprecated, pivot), the lightweight uninstall leaves lifeboat data, logbook reports, and visibility-driven gitignore entries that no longer make sense. `destroy` is the nuclear option — explicit, confirmed, complete. Two-tier on the same verb (`ahoy`) keeps the naming surface small while making the danger gradient legible.

## What's In Scope

- `/abcd:ahoy destroy` sub-verb (two-tier with the `uninstall` sub-verb)
- Hard confirmation showing exact list of files/directories to delete
- Revert `.gitignore` allowlist entries added by ahoy (back to project's own state)
- Removes scheduled `dev-sync` jobs if installed (forward-compatible with itd-13)
- Final summary report
- Documentation update: `04-surfaces/01-ahoy.md` documents both sub-verbs side-by-side with the danger gradient explicit (and `uninstall`'s description rewords to "reversible marker-only removal" so the distinction is discoverable from bare `/abcd:ahoy` status+help output).

## What's Out of Scope

- `/abcd:purge` as a separate command — explicitly rejected. Two sub-verbs on one command (with a `destroy` that names its own danger) keeps the surface small. The earlier `/abcd:purge` working name is preserved here for git-blame continuity, but the shipping name is the `destroy` sub-verb on `/abcd:ahoy`.
- "Soft destroy" (archive `.abcd/` to `.abcd-archive/` instead of delete) — a possible future extension
- Cross-project destroy ("remove abcd from all my repos") — requires global state we don't yet have
- Forced destroy without confirmation — `destroy` is destructive; always confirm

## Acceptance Criteria

> _Required (per the itd-1 discipline). At least one Given-When-Then bullet describing the verifiable bar for "shipped"._

- **Given** an abcd-installed repo with a populated `.abcd/` directory and ahoy-managed marker block in `CLAUDE.md`, **when** the user runs `/abcd:ahoy destroy`, **then** abcd shows a hard-confirmation prompt enumerating every file/directory queued for deletion (marker block, symlink, `.abcd/` namespace contents, `.gitignore` allowlist entries) before any deletion occurs.
- **Given** the user confirms the `destroy` prompt, **when** the command completes, **then** the repo no longer contains the abcd marker block in `CLAUDE.md`/`AGENTS.md`, the `/usr/local/bin/abcd` symlink is removed, the `.abcd/` directory is fully removed, and `.gitignore` ahoy-managed allowlist entries are reverted to their pre-ahoy state.
- **Given** the user declines the `destroy` confirmation, **when** the command exits, **then** no files or directories are modified and the command's exit code reflects the user's cancellation (non-zero, distinct from error codes).
- **Given** scheduled `dev-sync` jobs were installed (per itd-13), **when** `/abcd:ahoy destroy` runs successfully, **then** the scheduled jobs are also removed and the final summary report enumerates the removed schedule entries.
- **Given** a repo with a clean abcd installation, **when** the user runs `/abcd:ahoy uninstall` (marker-only) followed later by `/abcd:ahoy destroy`, **then** `destroy` still removes the residual `.abcd/` namespace contents that `uninstall` left behind — the two sub-verbs are composable, not mutually exclusive.

## Open Questions

- ~~Should `destroy` offer to back up `.abcd/lifeboat/` somewhere first (since it may be valuable independently)?~~ Moot under adr-35: a lifeboat is written out of tree, to a destination `disembark pack` is handed, so `destroy` finds none inside the repository.
- Same question for `.abcd/development/` (it's design history, not just runtime state)?
- **Naming — resolved 2026-05-07.** Two-tier on `/abcd:ahoy`: `/abcd:ahoy uninstall` is reversible marker-only removal (re-running `ahoy` re-installs cleanly); the deeper destroy surfaces as `/abcd:ahoy destroy` — fits the nautical metaphor (scuttling a ship has a name and a weight) and the verb's danger is encoded in the verb itself. The `uninstall` description ("reversible marker-only removal") makes the distinction discoverable. Plan-review for this intent treats this naming as decided rather than re-opening.

## Amendments

Amended by itd-2610030814013772 (2026-10-03, its decisions 4 and 8): the marker block lives in AGENTS.md; a block an earlier setup planted in CLAUDE.md is still found and stripped, so the CLAUDE.md wording here describes projects set up before that change.

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._
