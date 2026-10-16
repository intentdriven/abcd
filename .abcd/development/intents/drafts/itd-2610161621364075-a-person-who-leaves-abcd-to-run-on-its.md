---
id: itd-2610161621364075
slug: a-person-who-leaves-abcd-to-run-on-its
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# Autonomous runs execute inside an OS-enforced sandbox

## Press Release

> A person who leaves abcd to run on its own (a drain, a build lane, or a headless agent abcd starts) can trust that the run cannot reach past its own work. Each autonomous run executes inside a sandbox the operating system enforces, not only abcd's command checks. Writes are limited to the run's worktree and abcd's home folder, and network access to origin and the forge. A run that finds no sandbox on the machine says so before it starts, and does not run unattended without one unless the person has waived it for that repository. abcd's command guard keeps teaching and refusing inside the sandbox, but it is steering, not the boundary: the research note of 2026-08-18 records that every vendor documents its command checks that way and puts the boundary in an OS sandbox or a throwaway environment. Open for planning: which mechanism on each platform (macOS Seatbelt, Linux Landlock or bubblewrap, or the host's own sandbox mode), and whether a missing sandbox blocks the run or warns.

## Why This Matters

> _Why this matters to the user — replace before planning._

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

_None recorded yet._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
