---
id: itd-2610030720038073
slug: abcd-keeps-its-home-folder-out-of-desktop-search-by-default
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# abcd keeps its home folder out of desktop search by default

## Press Release

> abcd keeps its home folder out of desktop search by default: on a Mac, Spotlight skips abcd's home (worktrees, run logs, transcripts, caches), so opening a lane's working copy no longer sets off an indexing burst, and a person who wants it searchable turns indexing back on with one setting

## Why This Matters

On 2026-10-02 an autonomous run opened eight lane worktrees in abcd's machine-scoped store under the person's home folder within a few minutes. The desktop indexer (Spotlight's indexing daemons, and its PDF importer) indexed them, and the machine's one-minute load rose from about 20 to 148 with no test running. Every lane waited on its load gate until the burst passed, roughly half an hour lost across the run (run log ~/.abcd/runs/488a0aa9/2026-10-02.jsonl, a `stop` line at 07:32Z). Each worktree is a full checkout that lives for hours and is then removed, so indexing it buys nothing a person searches for. The store will grow as the worktree store intent (itd-2609091014076309) ships.

Typed links: refines itd-2609091014076309 (the machine-scoped worktree store whose creation set off the burst); refines the principle `the-users-directory-is-theirs` (the home is abcd's declared space). The rule that abcd achieves this only by changing its own folder, never the computer's search settings, is routed to its own decision record (the itd-84 routing the product thinker confirmed on 2026-10-03).

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

- Method, to be settled against primary sources before planning: a per-directory marker file versus a folder name the indexer skips, which of them current macOS still honours, and whether one set on the home covers every folder beneath it.
- Whole home or scratch only: the sources library under the home holds documents a person may want to find through desktop search; does the default exclude it with everything else, or only the transient stores (worktrees, runs, transcripts, caches)?
- Existing installs: an already-created home is applied by the setup verbs (install and update), by the first write into the home, or only on request.
- Other platforms: Linux and Windows desktop indexers, in scope now or later.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
