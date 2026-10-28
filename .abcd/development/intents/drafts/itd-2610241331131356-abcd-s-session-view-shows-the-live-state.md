---
id: itd-2610241331131356
slug: abcd-s-session-view-shows-the-live-state
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610170634394993]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
held: "held until itd-2610170634394993 (the native central session view) is planned: this adapter only adds to that view; routing confirmed by the facilitator 2026-10-24"
---

# The session view reads live agent state from a workspace manager

## Press Release

> abcd's session view shows the live state of each agent session (working, blocked, done, idle) that a terminal workspace manager the person installed reports, alongside the native session view. The adapter is read-only and opt-in: abcd never installs, starts, configures or depends on such a tool, never lets it place worktrees, and falls back to the native view when none is present. Reframed from a private ideate verdict of 2026-10-24 that rejected making such a tool the default environment for abcd-managed repos.

## Why This Matters

A person running many agent sessions on one repository cannot tell which are working, which are waiting on them, and which have stopped. On 2026-10-24 eight sessions ran against this checkout at once. A branch sweep could not tell a live session's new worktree from an abandoned one, because nothing links a worktree to its session (iss-2608220750029993).

The native central view (itd-2610170634394993) answers this across computers and accounts. A workspace manager the person already runs knows more about the sessions on one computer: the live state of each agent's terminal. Reading that state when it is present makes the view sharper for those people, and costs nothing for everyone else.

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
