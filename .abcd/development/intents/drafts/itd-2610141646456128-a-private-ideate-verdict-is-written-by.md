---
id: itd-2610141646456128
slug: a-private-ideate-verdict-is-written-by
spec_id: null
kind: bundle-member
bundle: ideate-librarian
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# A private ideate verdict stays private

## Press Release

> **A person who keeps an idea private can record its verdict without anything
> reaching the project.** `abcd ideate record --private` writes the verdict to
> the person's own store, readable by them alone, with no decision-log line. The
> source behind an idea is named publicly only once the idea is accepted, and
> accepting it adds the source's credit.
>
> "I test half-formed ideas from things I read," said Alice, a solo founder.
> "The killed ones and their sources stay on my machine. The one I adopted went
> public with its source credited, in the same change."

## Why This Matters

The product thinker ruled on 2026-10-14 that each ideate run asks whether the
idea is private or public, private verdicts going to
`~/.abcd.noindex/ideas/<root-sha>/` with no decision-log line. The binary cannot
do it, so every private run today records in a throwaway worktree and copies
the note out by hand (iss-2610090740139804); the lab of 2026-10-14 reproduced that, and
found the hand copy left the notes readable by others until fixed by hand. The
sibling rules wait on the same change: a public source named only on acceptance
(iss-2610090748299348) and its credit added on adoption (iss-2610090742488191).

Decided in the product thinker's planning interview of 2026-10-14 (session abcd-66), which followed lab lab-261014142726-9c4d759, a hand-run of /abcd:ideate on one external write-up.

## Mechanism

We expect privacy to hold because the binary itself chooses the store, the file
mode and whether a decision-log line is written, so no hand step can leak a
verdict or its source.

## Scope Conditions

- One person's machine; the private store is keyed on the repository's root
  commit like the other machine stores.

## Acceptance Criteria

- **Given** `--private`, **when** a verdict is recorded, **then** it is written
  under `~/.abcd.noindex/ideas/<root-sha>/` with owner-only permissions, no
  decision-log line is appended, and the working tree is unchanged.
- **Given** a private verdict, **when** its idea is accepted, **then** the
  graduating change names its public source and adds its credit to
  ACKNOWLEDGEMENTS.md in the same change.
- **Given** a public verdict for an idea that is killed, **when** it is recorded,
  **then** no public source behind it is named in the committed record.

## Open Questions

- What "accepted" means mechanically (a graduation to a draft intent, or a
  planned one) for naming a source (product thinker).

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
