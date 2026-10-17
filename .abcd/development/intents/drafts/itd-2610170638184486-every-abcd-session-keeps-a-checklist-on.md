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

> Every abcd session keeps a checklist on disk that says what it may decide, what it is waiting on and what it can do next, so an autonomous run never stalls and one stalled session never holds up the rest. Alice, a solo founder who is also the product thinker, leaves for the evening and tells their sessions to work on until 09:00. abcd records that as a mode with an owner and an end time, and each session's checklist carries it: Alice's design decisions that are signed off, the technical facilitator set to auto, the kinds of choice still raised with a person (security, performance, compatibility), the items waiting and on whom, the session's own promises ('when that pull request merges, do this'), and the work it owns, read from its claims, its pull requests and its branches. The checklist is repeated to the session at every start and after every compaction, so nothing that matters lives only in its context. A session that tries to stop while it has work it can do is handed the next item instead; when everything waits on something outside, it sets its next wake rather than going quiet. Before the run starts, while Alice is still there, each session writes into its checklist what could interrupt its part of the night (a design decision not yet made, a step only Alice's hands can do, work it depends on from another session, a credential it lacks), so the pass that coordinates the sessions can put them to Alice together and tell them whether the run is likely to stay autonomous. In the morning Alice reads two short lists: the steps only their own hands can do, and the decisions taken while they were away, each with its grounds and how to reverse it.

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

The product thinker added the next morning: before an autonomous run starts, or
as it starts, every session raises what may interrupt it and reports to the
orchestrating session, so the obvious blockers are raised while the product
thinker and the technical facilitator are still present. The orchestrator asks
every session for this, takes the answers to them together, and once every
session reports nothing left it tells them the run is likely to stay
autonomous. On the night of 2026-10-16 every blocker that later stopped work
was already known at 20:48, while the product thinker was still at the
terminal: the two GitHub Apps and their secrets, and the question about an
unowned pull request.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Scope Split

Agreed with the session drafting the coordination intent ("Headless agents in
parallel coordinate through the shared remote, not conversation", on branch
draft/conductor-intent until it lands; each record links the other once both
are on the default branch), and settled by the product thinker in that
session's interview on 2026-10-17:

- **This intent (one session's own state):** the checklist on disk and its
  re-injection at start and after compaction, the modes, the waits, the trigger
  promises, the stop check, the timed wake, the lists for the person's return,
  and the pre-run blocker sweep, which asks each session by message what could
  interrupt it (the product thinker chose this over moving the sweep into the
  coordination pass).
- **The coordination intent (state shared across sessions on the remote):**
  owned work with leases and take-over, so leases live in one place; the
  central view, one short status line per session (what it holds, its state in
  one line, what it waits on, when it was last seen), published only when it
  changes and only after the session has written its checklist; and each
  autonomous session's own credential.

## Decisions

The product thinker's answers of 2026-10-17, restated, and the agent's
technical decisions on state-of-the-art grounds, each marked.

- **Where the checklist lives:** on this computer only for now. A published
  option on GitHub (public) and a private shared option (for example through
  abcdev.app) are part of this work (product thinker).
- **What the private shared option shares:** everything by default,
  configurable later (product thinker).
- **The phone surface is part of this work.** An installable web app on the
  private shared option, rather than a native app, reusing the live brief
  session's relay and GitHub sign-in; a native app only if lock-screen answers
  or widgets prove necessary. An answer from the phone only chooses among the
  options offered or adds a note. Its look is chosen in design rounds in
  lab-261017071130-6fd3032 (product thinker for scope; agent for the form).
- **Who answers from the phone:** each person by their role: product questions
  to the product thinker, technical trade-offs to the technical facilitator
  (product thinker).
- **When the phone notifies:** configurable; by default only when a session's
  work would otherwise stop (product thinker).
- **The stop check:** a session that tries to stop while it has work it can do
  is refused and handed its next item only while the product thinker is
  recorded as away; while they are present the stop is allowed and flagged.
  The refusal is always made plain to them, so it never surprises them. A
  session genuinely waiting on something outside records what it waits for and
  when it will look again, and is never kept spinning (product thinker).
- **How a session learns the product thinker is away:** they say so, or set it
  in the app, either until a set time or open-ended; open-ended, they say when
  they are back. The state is always shown plainly, so they never wrongly
  believe agents are waiting or working (product thinker). Their stated goal,
  which the design must make possible: almost all work runs autonomously once
  the human design work is done.
- **The release:** this intent and the coordination intent form the
  autonomous-agents bundle, built as the next big release once current work
  settles; dashboard work joins later (product thinker).
- **Credentials:** autonomous sessions never use the product thinker's own
  forge login; each runs under its own weaker credential, owned by the
  coordination intent. The checklist records the credential's kind, scope and
  expiry, never its value or anything that retrieves it; a missing or expiring
  credential is a blocker-sweep question and owed work (product thinker, with
  the coordination session).
- **Crash safety:** the checklist survives a session killed mid-write, a host
  or terminal restart and a reboot, leaving the last complete checklist
  readable and re-injected, never truncated or empty (captured by the product
  thinker on the coordination session's branch; linked once it lands). The claim
  is confirmed only after a state-of-the-art pass and a lab that kills a session
  mid-write and reboots.
- **Format and re-injection (agent, on state-of-the-art grounds):** the
  checklist is JSON, not Markdown, because long-running agent harnesses find
  models overwrite Markdown state more readily; it is re-injected through the
  session-start hook's compact event, which the host re-reads after compaction.
  Writes use a temporary file, fsync, rename and a directory fsync, keep a
  last-good copy, and validate on read.
- **What the product thinker sees (agent):** a session's state shown to a
  person is derived from what the machine can see (its claim renewal, its
  process, its last commit); what the agent writes about itself is a hint,
  since agent-written status lags.

## Open Questions

_None for the product thinker at this point; the acceptance criteria are walked
in the planning interview._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
