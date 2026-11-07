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
- Holds for forge and CI events first. A chat channel is out of scope until a
  need for one shows.
- Holds where the wake carries only the event's kind, its ids and a link, never
  text from the event. Forge and CI text can be written by an outsider, so the
  session reads it itself, under the contract of itd-151 and the fence of
  itd-2610161621364075.
- Holds where deciding which events act and who owns them is code. A model may
  propose an action and never admits an event.
- Holds where an event wakes a session that is already running and never starts
  one, so adr-2610222222326104's first condition is untouched. With no session
  to wake, the event is left for itd-2611041656042921's notes.

## Acceptance Criteria

- **Given** a pull request behind its base with all checks green, **when** the
  watcher sees it, **then** the owning session is woken with the event and a
  proposed action, and no change is made by the watcher.
- **Given** no actionable event, **when** time passes, **then** no model runs.
- **Given** an event carrying text an outsider wrote, such as a pull request
  comment, **when** the watcher wakes the owning session, **then** the wake holds
  the event's kind, its ids and a link, and none of that text.
- **Given** an actionable event and no running session that owns it, **when** the
  watcher sees it, **then** no session or agent is started.

## Open Questions

- Which components are watched first, and how the owning session is found
  (technical facilitator).
- Whether the watcher shares the timer of itd-2611041656012738 or runs only while
  a session is alive (technical facilitator).

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
