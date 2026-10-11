---
id: itd-2610050548126044
slug: abcd-reminds-a-session-to-refresh-its
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

# abcd reminds a session to refresh its handover note only when that session changed the tree

## Press Release

> abcd reminds a session to refresh its handover note only when that session changed the tree. In a repository abcd manages, abcd's own plugin notes the working tree's state when a session starts and, when the session stops, asks it to update .abcd/.work.local/NEXT.md only if the tree changed since that start and the note is older than the session's work. Today the reminder is a personal hook outside abcd (~/.claude/hooks/stop-next-reminder.sh, dated 8 July) that fires whenever the tree is dirty and the note is more than two hours old, so a session that only read files is told to update the note because of untracked files an earlier session left behind (seen 2026-10-05). abcd already defines the note's place in the local tier and already runs a session-start hook, so it can tell this session's changes from leftovers, and every abcd user gets the reminder rather than one machine. The personal hook is removed once this ships. Draft acceptance criteria: a session that changed nothing is never reminded, whatever earlier sessions left in the tree; a session that changed files and left the note older than its first change is reminded once per stop and never loops; a repository abcd does not manage sees no reminder; the hook reads only the local tier and git status, writes nothing, and costs nothing on a prompt.

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
