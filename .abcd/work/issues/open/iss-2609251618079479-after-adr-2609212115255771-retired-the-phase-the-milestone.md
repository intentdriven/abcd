---
schema_version: 1
id: "iss-2609251618079479"
slug: "after-adr-2609212115255771-retired-the-phase-the-milestone"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/04-scope.md"
remedy: "Finish the sweep in the files it left, each once the branch editing it has landed: where the brief uses the phase or the milestone as the sequencing unit, state sequencing as dependencies (blocked_by, builds_on) plus the lifecycle shelves, the Now / Next / Later block and target_release, as adr-2609212115255771 decides; restate 04-surfaces/09-reflect.md at the release grain in the change that plans or builds itd-24; and ask the product thinker whether the build-sequence chapter's build-milestone sense retires with the phase. Grounds: the ADR fixes the replacement vocabulary, so each remaining site is a reading, not a decision, except the build-milestone sense, which the ADR does not name; a grep for phase and milestone over the brief that finds only history, the retired entries and the other senses listed in the record would show the sweep done."
deferred_after: "v0.11.1"
deferral_reason: "no ruling owed; carried past v0.11.1 by lane drainDrift3 (run A 2026-09-29) on its size: adr-2609212115255771 settles the model (sequencing is builds_on and blocked_by plus the drafts, planned and shipped shelves; the checkpoint is the derived release), but at 8322cdf65 the brief still names the phase at about 250 sites in 60 files, and each needs a reading to tell the retired sequencing unit from a legitimate use (the retired-term glossary entry, the phase-audit receipt, a process phase). It wants a docs lane of its own, taking 01-product/04-scope.md and the release, version, loop, plan and spec glossary entries first."
---

After adr-2609212115255771 retired the phase, the milestone and the word roadmap, the brief outside the mental-model chapter still describes the phase as the live sequencing unit: 59 files under .abcd/development/brief mention it, among them 01-product/04-scope.md (bounded by the planned phases), and the glossary entries release, version, loop, plan and spec, whose prose says a phase sequences the work and a release falls out of completing one. itd-2609211913453478 rewrote only the mental-model chapter and repointed each entry's not_to_be_confused_with; no record carries the rest of the sweep.

## Sweep progress 2026-09-29

Lane drainBrief (autonomous run A, c80832f97) swept the brief outside the files
other branches were editing. Done: `01-product/04-scope.md` (the scope is the
shelves, sequenced by dependencies and rendered as the Now / Next / Later
block), the glossary entries release, version, loop, plan and spec, and the
phase-as-sequencing sites in `01-product/` (press release, context, README),
`02-constraints/01-platform.md` and `02-dependencies.md`,
`03-evidence/04-tradeoffs.md`, `04-surfaces/02-disembark.md`, `03-embark.md`
and `07-memory.md`, `05-internals/` 01, 04, 05, 07, 08 and 09,
`06-delivery/README.md` and the opening of `01-build-sequence.md`, and the
glossary's core README, brief, reading-position and ledger/position entries
and the distribution README.

Left, and why:

- Files other branches were editing, which this lane was barred from:
  `02-constraints/04-naming.md` (15 sites), `02-constraints/03-invariants.md` (1),
  `05-internals/03-configuration.md` (7), `06-delivery/02-verification-matrix.md`
  (2), `06-delivery/03-out-of-scope.md` (22), `04-surfaces/01-ahoy.md` (1),
  `04-surfaces/04-launch.md` (4), `04-surfaces/05-intent.md` (36) and
  `04-surfaces/README.md` (1).
- `04-surfaces/09-reflect.md`: its whole design is phase-grained (argument,
  seed, output path). The chapter now says the grain is retired and that the
  release-grain design is owed with itd-24; restating it is a design act for
  that intent, not a sweep. The `reflection-composer` row of
  `05-internals/01-agents.md` (from a phase-audit receipt) goes with it.
- `06-delivery/01-build-sequence.md`: its build milestones are the brief's
  plumbing sequence for the Go core, a sense the phase glossary entry keeps
  apart and the ADR does not name. Only its opening pointer at the phase
  documents is swept.

Kept on purpose, as history or as another sense of the word: the retired
entries `glossary/core/phase.md`, `milestone.md` and `roadmap.md`;
`introduced_in: phase-N` provenance; links to the phase documents as history
(`glossary/core/loop.md`, `glossary/core/ledger.md`); `00-meta.md:77`, whose
source list mirrors `internal/core/lifeboat/mapping.go:206`; the process phases
of `04-surfaces/15-prepare-this-repo.md` and the grill session's phases under
`glossary/interview/`; and the voyage entry's lifecycle table.
