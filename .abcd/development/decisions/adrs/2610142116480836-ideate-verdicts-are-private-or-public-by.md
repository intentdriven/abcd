---
id: adr-2610142116480836
slug: ideate-verdicts-are-private-or-public-by
status: accepted
date: 2026-10-14
supersedes: null
superseded_by: null
related_intents: [itd-2610141646456128, itd-2610141646452213, itd-2610141646454347]
related_rfcs: []
related_adrs: [adr-41, adr-2610141646525117]
---

# ADR-2610142116480836: Ideate verdicts are private or public by the person's choice, and an idea's source is named publicly only when it graduates

## Context

`abcd ideate record` writes every verdict into the repository: a research note
and a decision-log line, both committed (iss-2610090740139804). The product
thinker treats ideas as private by default and ruled on 2026-10-09 that a public
source behind an idea is named only once the idea is accepted
(iss-2610090748299348), and on 2026-10-14 that each ideate run asks whether the
idea is private or public. Until the binary can write privately, private runs
record in a throwaway worktree and copy the note out by hand, and a lab of
2026-10-14 found that hand copy left notes readable by others. In the same
interview the product thinker ruled that a verdict whose idea came from a private
reading pack is always private.

## Decision

- **Every verdict is private or public by the person's choice, never by
  default.** `ideate record` takes `--private` or `--public`; with neither, it
  asks at a terminal and refuses elsewhere, writing nothing.
- **A private verdict stays on the person's machine:** under
  `~/.abcd.noindex/ideas/<root-sha>/`, owner-only, through the same redaction as
  a public one, with no decision-log line. It may be cited by another private
  verdict, never by a public one.
- **A verdict whose idea came from a private reading pack is always private.**
- **An idea's origin source is named publicly only when the idea graduates to a
  draft intent**, and the graduating change adds its credit to
  ACKNOWLEDGEMENTS.md. A public verdict on a killed idea leaves its origin
  sources out of the committed record; each claim's primary source stays. A
  source held in the private sources corpus is named publicly only after a
  person's ledger flip (adr-41).

## Alternatives Considered

- **Private by default, public on request:** never leaks by omission, but
  contradicts the ruling that each run asks. Rejected.
- **Public by default, private on request:** today's behaviour plus a flag; a
  forgotten flag commits a private idea. Rejected.
- **Name sources once an intent is planned, not drafted:** later and safer, but
  the two drafts that graduated on 2026-10-14 already named theirs at the draft.
  Rejected.

## Consequences

- An unattended run that cannot ask must say private or public up front.
- A private verdict is invisible to teammates and to the repository's gates; its
  only reader is the person and their own later ideate runs.
- The rule binds itd-2610141646456128 (the store and the naming rule) and
  itd-2610141646452213 (verdicts from private packs).
