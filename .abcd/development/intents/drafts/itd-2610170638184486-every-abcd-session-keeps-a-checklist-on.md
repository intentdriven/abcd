---
id: itd-2610170638184486
slug: every-abcd-session-keeps-a-checklist-on
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# Every session keeps a checklist on disk, so no run stalls and no stall spreads

## Press Release

> Every abcd session keeps a checklist on disk that says what it may decide, what it is waiting on and what it can do next, so an autonomous run never stalls and one stalled session never holds up the rest. Alice, a solo founder who is also the product thinker, leaves for the evening and tells their sessions to work on until 09:00. abcd records that as a mode with an owner and an end time, and each session's checklist carries it: Alice's design decisions that are signed off, the technical facilitator set to auto, the kinds of choice still raised with a person (security, performance, compatibility), the items waiting and on whom, the session's own promises ('when that pull request merges, do this'), and the work it owns, read from its claims, its pull requests and its branches. The checklist is repeated to the session at every start and after every compaction, so nothing that matters lives only in its context. A session that tries to stop while it has work it can do is handed the next item instead; when everything waits on something outside, it sets its next wake rather than going quiet. Work a session owns carries a lease like a claim, so when one session stops answering, another takes its items over and says so, and no lock, queue or release role waits on a silent session. In the morning Alice reads two short lists: the steps only their own hands can do, and the decisions taken while they were away, each with its grounds and how to reverse it.

## Why This Matters

_Written by the agent on 2026-10-17 from the overnight run of 2026-10-16 and the
product thinker's framing that morning; the planning interview confirms or
rewrites it._

On the night of 2026-10-16 every session had been told to work autonomously
until the morning, and one session sat idle for about ten of eleven and a half
hours while it held work it could do alone; the coordinating session held three
finished, reviewed steps behind a merge order that only fixed when they land.
The record of it is iss-2610170619247185. Everything that mattered lived in
the sessions' context: the instruction to work on was relayed by a peer,
overridden in one session's reading when the product thinker briefly came back,
and dropped from the summary when the context was compacted. A promise ("when
that pull request merges, add the wrapper") lived in a watcher that ended after
two hours. Nothing in abcd saw the stall, and a Claude Code session runs only
when something sends it input.

The product thinker's framing, restated:

- Their essential design decisions are the ones with real trade-offs and more
  than one valid answer; obvious answers are decided by the agent and recorded.
  A session must be able to tell that the essential design work for a feature
  is truly done.
- The technical facilitator can be set to auto, in which case the agent takes
  technical decisions on its own; otherwise genuine trade-offs (security,
  performance, compatibility) are raised with them and the rest are taken and
  recorded.
- Implementation is always autonomous, with its decisions and issues reviewed
  and recorded.
- Nothing stalls: while an item waits, the session keeps itself busy with work
  it can do.
- One stalled session never holds up the others.

The overnight run added: a standing instruction needs an owner and an end time;
a promise tied to a trigger needs to be written down; waiting is per item, never
per session, and stacked work proceeds while only its landing order waits; a step
only a person's hands can do (creating an App, setting a secret) is a third kind,
collected for their return; decisions taken while a person is away are listed for
them with grounds and how to reverse; a list cannot help a session that never
runs, so stopping needs a check and waiting needs a timed wake; and a
coordinator needs to see a stall.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

_Seeded by the agent on 2026-10-17; each lists the defensible answers, none
recommended. Everything else in this draft is the agent's to decide and record._

- **Where does the checklist live?** (a) on this computer only, in the
  checkout's local tier; (b) published on origin, like reservations, so every
  session, every computer and the second contributor see it; (c) the modes and
  the lists for the person's return published, each session's own working
  items local.
- **How hard does the check at a session's stop bite?** (a) it refuses the stop
  and names the next item; (b) it lets the stop through with a warning that the
  coordinator and the status line see; (c) it refuses only during an autonomous
  run and warns otherwise.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
