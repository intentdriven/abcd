---
id: itd-2610141646469491
slug: a-cheap-event-watcher-follows-one
spec_id: null
kind: standalone
bundle: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# A cheap watcher wakes an agent only when something needs action

## Press Release

> **A stuck pull request or a failed queue entry wakes an agent within minutes,
> and nothing runs a model while all is well.** A cheap script follows one
> component's events and wakes the coordinating agent only on an actionable
> event. The agent reports and proposes, and acts only through the session that
> owns the change.
>
> "A pull request sat behind its base for an afternoon with every check green,"
> said Frank, an SRE. "Now the watcher wakes the session that owns
> it, which updates the branch, and no model sits refreshing charts in between."

## Why This Matters

Survived, reframed, from an ideate run of 2026-10-14 (private verdict
component-watcher-agents) on the engineering write-up "Redesigning source control at Anthropic" (Anthropic, 2026-10-13, <https://www.anthropic.com/engineering/redesigning-source-control>), whose "superfan" agents each watch one
component's telemetry and logs. The write-up gives no measurement of their
effect, and a standing model per component spends tokens and sub-agent slots
while idle; both reviewers of the run reframed it to an event-triggered cheap
watcher, the shape an earlier private verdict (coordinator-crew, 2026-10-09)
had also reached. The real cost it targets is on the record: pull requests
wedged behind their base (iss-172).

## Mechanism

We expect faster recovery at almost no cost because the watcher is a script,
not a model, and the agent is woken only on an event that needs action, so the
expense scales with incidents, not with time.

## Scope Conditions

- The watcher acts on nothing itself; changes go through the owning session,
  under the concurrency rules (scan, announce, the cutting session's say).

## Acceptance Criteria

- **Given** a pull request behind its base with all checks green, **when** the
  watcher sees it, **then** the owning session is woken with the event and a
  proposed action, and no change is made by the watcher.
- **Given** no actionable event, **when** time passes, **then** no model runs.

## Open Questions

- Which components are watched first, and how the owning session is found
  (technical facilitator).

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
