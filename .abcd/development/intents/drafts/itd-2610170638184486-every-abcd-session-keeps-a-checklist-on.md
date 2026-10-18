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

> Every abcd session keeps a checklist on disk that says what it may decide, what it is waiting on and what it can do next, so an autonomous run never stalls and one stalled session never holds up the rest. Alice, a solo founder who is also the product thinker, leaves for the evening and tells their sessions to work on until 09:00. abcd records that as a mode with an owner and an end time, and each session's checklist carries it: Alice's design decisions that are signed off, the technical facilitator set to auto, the kinds of choice still raised with a person (security, performance, compatibility), the items waiting and on whom, the session's own promises ('when that pull request merges, do this'), and the work it owns, read from its claims, its pull requests and its branches. The checklist is repeated to the session at every start and after every compaction, so nothing that matters lives only in its context. A session that tries to stop while it has work it can do is handed the next item instead; when everything waits on something outside, it sets its next wake rather than going quiet. Before the run starts, while Alice is still there, each session writes into its checklist what could interrupt its part of the night (a design decision not yet made, a step only Alice's hands can do, work it depends on from another session, a credential it lacks), so the orchestrating session can put them to Alice together and tell them whether the run is likely to stay autonomous. In the morning Alice reads two short lists: the steps only their own hands can do, and the decisions taken while they were away, each with its grounds and how to reverse it.

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

_Draft for the product thinker to walk in the planning interview: derived by
the agent on 2026-10-18 from the press release and the decisions below, none
of it yet confirmed. The away mode, the stop check and the question reroute
are criteria of itd-2610180912189125, the slice that builds on this draft,
and are not repeated here. The last three move with the phone and the shared
options if the split proposed for Thursday is confirmed._

- **Given** Alice, the product thinker, has said the sessions work on until
  09:00, **when** a session writes its checklist, **then** the file in its
  working copy's local tier holds the design decisions signed off, whether
  the technical facilitator is set to auto, the kinds of choice still raised
  with a person, each item waiting and on whom, its trigger promises, and the
  work it owns, read from its reservations, its pull requests and its
  branches.
- **Given** a session with a checklist, **when** it starts or its context is
  compacted, **then** the last complete checklist is added to its context by
  the session-start hook, the compact source included.
- **Given** a session killed while writing its checklist, or a reboot,
  **when** the next session starts in that working copy, **then** it reads and
  is given the last complete checklist, never a truncated or empty one, and an
  unreadable file falls back to the last good copy and says so.
- **Given** a checklist promise "when pull request 42 merges, add the
  wrapper", **when** the session next looks (at a start, a wake or a stop
  check) after 42 has merged, **then** the promise is named as its next item,
  with no watcher process kept running.
- **Given** a session whose items all wait on something outside, **when** it
  records the wait, **then** its checklist names what each item waits for and
  when it looks again, sooner than half its reservations' lease, and its state
  reads waiting, never stopped.
- **Given** Alice is about to leave, **when** the orchestrating session runs
  the blocker sweep, **then** every session writes into its checklist any
  undecided design decision, step only a person's hands can do, work it
  depends on from another session, and credential missing or expiring, the
  orchestrator puts them to Alice together, and it says the run is likely to
  stay autonomous only once every session reports none open.
- **Given** the sweep finds a credential that expires before the window ends,
  **when** the session records it, **then** the checklist holds its kind, its
  scope and its expiry, and never its value or anything that retrieves it.
- **Given** Alice comes back, **when** they read what the sessions left them,
  **then** they see two short lists across every session: the steps only
  their own hands can do, and the decisions taken while they were away, each
  with its grounds and how to reverse it.
- **Given** Nia, the technical facilitator, is set to auto, **when** a session
  meets a technical trade-off, **then** it decides and records it; **and
  given** Nia is set to a person and is present, **when** the trade-off is one
  of security, performance or compatibility, **then** it is raised with them,
  and any other is decided and recorded.
- **Given** a feature whose design trade-offs are listed in its checklist,
  **when** a session asks whether the human design work for it is done,
  **then** the answer is yes only when every genuine trade-off has a recorded
  answer from the product thinker.
- **Given** a session's state shown to a person, **when** what the session
  wrote about itself disagrees with what the machine can see (its claim
  renewal, its process, its last commit), **then** the state shown follows
  the machine, with the session's own words beside it as a hint.
- **Given** the private shared option is on, **when** a session writes its
  checklist, **then** the checklist is shared in full by default, and a person
  can narrow what is shared.
- **Given** the public option is on, **when** it publishes a checklist,
  **then** nothing carrying the id of an advisory not yet published leaves the
  machine, by the same check announcements apply.
- **Given** the phone app on the private shared option, **when** a session's
  work would otherwise stop on a question, **then** the person whose role it
  belongs to (product questions to Alice, technical trade-offs to Nia) is
  notified, and can only choose among the options offered or add a note.

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
- **The first slice (itd-2610180912189125, which builds on this draft):** the
  away mode held by abcd and repeated at start, compaction and prompt, the
  stop check, and the question reroute, carved out on 2026-10-18 as the fix
  for iss-2610170619247185 so the security and bug-fix release waits on it
  alone (dcn-2610180913379049, dcn-2610180913388445). Its away mode grants
  no rights; the person's standing instruction decides what is pushed
  (dcn-2610181016510153). This draft keeps the rest and reuses the slice's
  away record and hook path.
- **Moved back from the first slice by its review round 1 (2026-10-18,
  dcn-2610180937181929), named remaining work of this draft:**
  - the report shown once on the first prompt after the away mode ends: the
    queued steps, the decisions recorded in the window and the stops the
    host forced, with who ended the mode and from which working copy;
  - the window's decisions gathered from every working copy, by `abcd
    decide` appending each decision's id to the away record under its lock
    while the window is open, since sibling worktrees hold their own
    decision records;
  - the status line naming the away mode and its end;
  - a bare `abcd mode` printing the away record, the queue and the owed
    items with no plugin host;
  - the brief's surface text for these.

  The slice keeps `abcd mode back` printing the queue and the recorded stops,
  one plain line each, so nothing queued goes unread meanwhile.
- **Moved back from the first slice by its trim after review round 2
  (2026-10-18, dcn-2610181016510153 and dcn-2610181017056307, on the release
  orchestrator's ruling), named remaining work of this draft:**
  - making the away mode the person's act alone, should a later design give
    it rights worth guarding: the guard entry refusing an agent's `abcd mode
    autonomous` (the binary on PATH and the plugin root's), and its routes
    through `sh -c` and a script file a shell runs; the shell-write refusal
    on the away store under `~/.abcd.noindex/away/`; the file-tool refusal,
    with the PreToolUse matcher gaining `Edit`, `Write`, `MultiEdit` and
    `NotebookEdit`; the refused attempts recorded in the store, shown on the
    person's next prompt and listed by `abcd mode back`; and the live checks
    that the person's `!` command reaches no pre-tool hook and that a
    message relayed from another session starting with `!` is not run as a
    shell command. The slice's away mode grants no push rights, and the
    person's standing instruction decides what is sent, so none of this
    holds up the stall fix; and since the OS fence (spc-2610161751056096)
    covers only runner-launched agents, it stays a mistake filter in the
    person's own sessions;
  - the guard reading `go run ./cmd/abcd` (go's flags stepped over, an
    `@<version>` allowed) and a built `abcd-<goos>-<arch>` as `abcd`, for
    every entry that names the program, `abcd-source-ledger-flip` included,
    which commands/guard.md today names as unseen. It widens what every
    entry matches, so it gets its own security review here rather than
    riding on the cut;
  - a reuse cache for the away stop check's forge listing (a listing younger
    than ten minutes reused, and on a failed read a cached listing younger
    than half the default lease used with its age named), so a run of blocks
    costs one call; the slice makes one uncached listing per check under a
    ten-second deadline;
  - anything `abcd mode back` prints beyond one plain line per queued step
    and recorded stop: grouping, formatting or a summary of the window.

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
- **A session's state (agent, matched with the coordination intent's
  central-view line):** one of working; waiting, with what it waits on and
  when it looks again; or stopped, which is allowed only while the product
  thinker is present and is flagged. While the product thinker is away a
  session is never stopped with work it can do, so a waiting session never
  reads as idle.
- **The timed wake keeps reservations alive (agent, with the coordination
  session):** a waiting session's next wake comes sooner than half the lease
  of the reservations it holds (under an hour for the two-hour lease), so an
  idle waiter's holds are renewed rather than lapsing and being taken over.
- **A public upload passes the embargo check (agent, on brief invariant 22):**
  if the public GitHub option is kept, nothing it publishes may carry the id of
  an advisory not yet published, the same check announcements apply
  (itd-2609151838312703).
- **What the product thinker sees (agent):** a session's state shown to a
  person is derived from what the machine can see (its claim renewal, its
  process, its last commit); what the agent writes about itself is a hint,
  since agent-written status lags.

## Open Questions

_None for the product thinker at this point; the acceptance criteria are walked
in the planning interview._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
