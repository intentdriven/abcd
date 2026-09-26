---
schema_version: 1
id: "iss-2609110944498549"
slug: "ahoy-install-must-not-write-abcd-by-name-into-a-target-repos"
severity: "major"
category: "ux"
source: "user-observation"
found_during: "v0.8.0 release review, maintainer ruling"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/defaults/claude-md-marker-block.md"
resolution: "The docs target defaults to skip, so a default ahoy install writes the managed, abcd-naming rule-loader block into neither CLAUDE.md nor AGENTS.md; a project that names claude_md, agents_md or both gets it there, and that choice is the approval to plant it. The adopted repo classifies as managed on its registry entry. An unset docs.target previews the default, so the dry-run no longer promises a block. prepare-this-repo's nameless acceptance criterion is restored and the v0.8.0 prose correction reversed. Scope: the conventions-file block this record's found_at and every settle-item address. Acceptance 1 is NOT met: after a default install, committed files still contain abcd, because the name-guard hooks (.githooks/pre-commit, .githooks/pre-merge-commit) and the .gitignore fence the install commits still name it. That remainder was carried, at this record's major severity, by iss-2609231103413459, and the product thinker ruled it a sanctioned exception on 2026-09-23 (ruling F, recorded in DECISIONS.md): the two name-guard hooks and the .gitignore fence keep their markers and naming as the one allowed mention of abcd in an adopted repository, with no rename and no migration, so acceptance 1 holds with that exception stated. The ruling covers the markers, not abcd's record ids, which the hook templates no longer cite; iss-2609231103413459 closed as wontfix on the ruling."
impact: fix
resolved_by:
  commit: "dae705d5"
---

**Maintainer ruling, 2026-09-11: `ahoy install` must not write abcd by name into
a repository it adopts.** This record exists to carry that decision to the change
that implements it.

## What happens today

`ahoy install` defaults `--docs-target` to `both`
(`internal/core/ahoy/detect.go:28`), and `stepMarker`
(`internal/core/ahoy/apply.go:866-900`) plants
`internal/core/ahoy/defaults/claude-md-marker-block.md` into the target
repository's committed `CLAUDE.md` **and** `AGENTS.md`. That block opens
"Managed by abcd (Agent-Based Configuration for Development)" and carries a full
"## abcd rule loader" section.

So a default adoption commits abcd's name and its internals into somebody else's
repository, and does so in files that repository's contributors read first.

## Why it is wrong rather than merely surprising

`prepare-this-repo` promises the opposite in its own acceptance criteria: "Given
the adoption completes, then nothing from `private-names.txt` and no
abcd-internal content appears in any committed artefact", reinforced by a
Boundaries claim that the output is nameless and never mentions abcd.

The v0.8.0 release gate caught the contradiction and the prose was corrected to
describe what the code does
(`.abcd/development/brief/04-surfaces/15-prepare-this-repo.md`). That was the
right move for the record at the time and is now the wrong end to have fixed:
the ruling is that the PROMISE was correct and the BEHAVIOUR should change.
Implementing this therefore reverses that prose correction, deliberately.

It is the same stance as
[`the-users-directory-is-theirs`](../../../development/principles/the-users-directory-is-theirs.md)
and adr-2609091248200336, one level in: a tool does not create directories in
space the user did not hand it, and it does not write its own name into files
the user will commit under their project's history either.

## What the change has to settle

- **The marker still has to be findable.** Detection promotes a folder to
  managed-repo on a marker block (`internal/core/ahoy/detect.go:139`,
  `strong := registered || markerFired`), so whatever replaces the named block
  must still be recognisable to `classify` and must not silently downgrade every
  adopted repo to unmanaged.
- **The rule loader has to keep working.** The block is not decoration; it is how
  a session learns the loader exists. If the committed half goes nameless, the
  loader's documentation has to live somewhere the adopting project chose.
- **The default is the question, not just the text.** `--docs-target both` is
  what makes this the out-of-the-box behaviour. Options include defaulting to
  `skip`, keeping the block but stripping the name and the internals, or moving
  the whole thing under `.abcd/` where the adopter already accepted a namespace.
- **Registry-only adoption already exists** as the other strong signal, so a
  nameless install is not a new mechanism, only a different default.

## Acceptance

- **Given** a repository adopted with default options, **when** the install
  completes, **then** no committed file in that repository contains the string
  "abcd" as a result of the install.
- **Given** that same repository, **when** `ahoy` classifies it afterwards,
  **then** it still reports managed-repo.
- **Given** the `prepare-this-repo` acceptance criterion about nameless output,
  **when** the change lands, **then** the criterion is true of the code and the
  v0.8.0 prose correction is reversed in the same diff.

## Grounds

- pursued: a default adoption leaves the conventions files byte-identical and still classifies managed-repo; shown wrong if a default install plants the block, if an adopted repo reads unmanaged, or if a first install naming a target plants nothing
