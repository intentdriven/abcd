---
id: itd-2610170638184486
slug: every-abcd-session-keeps-a-checklist-on
spec_id: spc-2610230805429429
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
related_issues: [iss-2610170619247185, iss-2610222026223032]
related_intents: [itd-2610170634394993, itd-2610180912189125, itd-2610222029228127]
severity: minor
origin: researcher-authored
production_mode: hand-written
impact: additive
---

# Every session keeps a checklist on disk, so no run stalls and no stall spreads

## Press Release

> Every abcd session keeps a checklist on disk that says what it may decide, what it is waiting on and what it can do next, so an autonomous run never stalls and one stalled session never holds up the rest. Alice, a solo founder who is also the product thinker, leaves for the evening and tells their sessions to work on until 09:00. abcd records that as a mode with an owner and an end time, and each session's checklist carries it: Alice's design decisions that are signed off, the technical facilitator set to auto, the kinds of choice decided overnight and listed for the morning (security, performance, compatibility), the items waiting and on whom, the session's own promises ('when that pull request merges, do this'), and the work it owns, read from its claims, its pull requests and its branches. The checklist is repeated to the session at every start and after every compaction, so nothing that matters lives only in its context. A session that tries to stop while it has work it can do is handed the next item instead; when everything waits on something outside, it sets its next wake rather than going quiet. Before the run starts, while Alice is still there, each session writes into its checklist what could interrupt its part of the night (a design decision not yet made, a step only Alice's hands can do, work it depends on from another session, a credential it lacks), so the orchestrating session can put them to Alice together and tell them whether the run is likely to stay autonomous. In the morning Alice reads two short lists: the steps only their own hands can do, and the decisions taken while they were away, each with its grounds and how to reverse it.

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

_Facilitator-proposed on 2026-10-22; signed off by the product thinker on 2026-10-23 (dcn-2610230805325396)._

We expect autonomous sessions to stop stalling and to come back from a
compaction with their work intact, because everything a session now keeps
only in its context (what it owns, what each item waits on, the promises it
made, when it looks again) is written to a file the host re-injects after
every start and compaction, and a session that tries to stop is shown that
file's next item. Shown wrong if, over a measured run of autonomous sessions
across at least one night and three automatic compactions, the share of
hours a session sits idle while holding work it could do, and the number of
items a session loses or repeats after a compaction, are no lower than in
the same run with only the stall fix (itd-2610180912189125).

## Scope Conditions

_Facilitator-proposed on 2026-10-22; signed off by the product thinker on 2026-10-23 (dcn-2610230805325396)._

- Holds for sessions on a host whose hooks fire at session start (with the <!-- cond: cond-2610230805423626 -->
  compact source), at prompt submission and at stop, and can add text to the
  model's context; Claude Code is the first such host. On a host without
  them, the checklist is kept but nothing re-injects it.
- Holds for a checklist that fits the host's limit on injected text (10,000 <!-- cond: cond-2610230805426039 -->
  characters on Claude Code, 2026-10-22); a longer checklist is cut to its
  open items, newest first, and says what it left out.
- Holds for one working copy per checklist on one computer; the copies <!-- cond: cond-2610230805429755 -->
  beyond it are itd-2610222029228127.
- Holds while the session itself writes the checklist as it works: no hook <!-- cond: cond-2610230805429053 -->
  can make a session act before an automatic compaction, so an item the
  session never wrote down is not recovered.

## Acceptance Criteria

_Draft for the product thinker to walk in the planning interview: derived by
the agent on 2026-10-18 from the press release and the decisions below, and
revised on 2026-10-22 for the product thinker's answers of that night and
iss-2610222026223032; walked and kept in full by the product thinker on 2026-10-23 (dcn-2610230805325396). The away mode, the stop check
and the question reroute are criteria of itd-2610180912189125, the slice that
builds on this draft, and are not repeated here. The three criteria for the
shared copies and the phone moved to itd-2610222029228127 on 2026-10-22
(dcn-2610222025212338)._

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
- **Given** an autonomous session partway through a multi-step plan (lanes
  dispatched, pull requests it watches, a promise to a peer, its next step),
  **when** the host compacts its context automatically, **then** the session
  resumes from the checklist it wrote before the compaction, naming the same
  next step and the same open items, with nothing it had written down lost
  (iss-2610222026223032).
- **Given** a session that owns a reservation, pull request or branch its
  checklist does not list, or whose owned work changed after the checklist
  was last written, **when** it ends a turn while the person is away, or the
  person next writes to it while present, **then** it is told its checklist
  is behind and which of those it does not list, so the file is current
  before any compaction can come. A step that left no such trace cannot be
  seen, and the scope conditions say so.
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
- **Given** Alice says she is away until 09:00, **when** the away mode is
  set, **then** the blocker sweep runs at once: every session writes into its
  checklist any undecided design decision, step only a person's hands can do,
  work it depends on from another session, and credential missing or
  expiring; the orchestrator puts them to Alice together; and the unattended
  run starts only once she has answered them (dcn-2610222025217495).
- **Given** the sweep finds a credential that expires before the window ends,
  **when** the session records it, **then** the checklist holds its kind, its
  scope and its expiry, and never its value or anything that retrieves it;
  a write that carries a secret is refused, naming the field without echoing
  it (adr-2610222026265719, brief invariant 24).
- **Given** Alice comes back, **when** they read what the sessions left them,
  **then** they see two short lists across every session: the steps only
  their own hands can do, and the decisions taken while they were away, each
  with its grounds and how to reverse it.
- **Given** Nia, the technical facilitator, is set to auto, **when** a session
  meets a technical trade-off, **then** it decides and records it; **and
  given** Nia is set to a person and is present, **when** the trade-off is one
  of security, performance or compatibility, **then** it is raised with them,
  and any other is decided and recorded; **and given** the person is away,
  **when** a session meets such a trade-off, **then** it decides it and lists
  it, with its grounds and how to undo it, for the person's return
  (dcn-2610222025225277).
- **Given** a feature whose design trade-offs are listed in its checklist,
  **when** a session asks whether the human design work for it is done,
  **then** the answer is yes only when every genuine trade-off has a recorded
  answer from the product thinker.
- **Given** a session's state shown to a person, **when** what the session
  wrote about itself disagrees with what the machine can see (its claim
  renewal, its process, its last commit), **then** the state shown follows
  the machine, with the session's own words beside it as a hint.

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

- **The shared copies and the phone (itd-2610222029228127, split out on
  2026-10-22):** the private shared option, the public option and the phone
  app, with the decisions about them below, which now bind that draft
  (dcn-2610222025212338).
- **Moved out to itd-2610240850336695 (the person's grant, 2026-10-24).** The
  parked person-only work above (the guard refusing a session's own `abcd mode
  autonomous` by every route, the shell and file-tool refusals on the away
  store, the wider PreToolUse matcher, the recorded attempts and the two live
  checks) is that intent's Decision 5, because it gives the away mode the rights
  worth guarding. That intent `builds_on` this one: when this draft's pre-run
  blocker sweep ships, it runs before an away grant takes effect, so the person
  is asked every undecided design decision first (its Decision 16).

## Decisions

The product thinker's answers of 2026-10-17, restated, and the agent's
technical decisions on state-of-the-art grounds, each marked. The first
five bullets now bind itd-2610222029228127, where they are carried.

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
- **The bundle gains a third member (product thinker, 2026-10-24).** The
  person's grant, itd-2610240850336695, joins the autonomous-agents bundle, so
  the grant, the checklist and the coordination intent are planned together.
  Its Decision 14 narrows "they say so" above: the person sets the away mode
  and the grant by the explicit command, never by a sentence in chat.
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

The product thinker's answers of 2026-10-22, restated, each a decision
record:

- **The phone and the shared copies are their own intent**
  (dcn-2610222025212338): not asked, because the answer that full uploads
  come later (dcn-2610222025114805) settled it.
- **The blocker sweep runs when the person says they are away**, and the run
  starts after they answer (dcn-2610222025217495).
- **Overnight trade-offs of security, speed and compatibility are decided
  and listed** with how to undo each (dcn-2610222025225277).
- **A checklist never holds a credential's value** (dcn-2610222025224993),
  written up as adr-2610222026265719 and brief invariant 24.
- **Waiting holds up one item, never a whole session**
  (dcn-2610222025230680), written up as the principle
  `waiting-holds-one-item-never-a-session`.

The facilitator's decisions of 2026-10-22, on state-of-the-art grounds:

- **No hook makes a session write its checklist before a compaction**
  (Claude Code's hooks reference, read 2026-10-22: the PreCompact hook can
  run a command or block the compaction but cannot make the model act, and
  PostCompact has no decision control). iss-2610222026223032's remedy asks
  the PreCompact hook to make the session bring its list up to date; that
  is not available, so the checklist is kept current as the session works,
  the stop check tells a session at each turn's end when its checklist is
  behind, and the session-start hook's compact source re-injects it.

The product thinker's walk and sign-off of 2026-10-23 (dcn-2610230805325396):

- **Signed off as written, with one wording fix.** The eleven stories stand, the technical list is accepted, and the press release, grounds, scope conditions and build order are signed off. The press release's second part now names the kinds of choice decided overnight and listed for the morning, not those still raised with a person, matching the product thinker's ruling of 2026-10-22 that such trade-offs are decided and listed; the product thinker signed off the corrected text.

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in
the [autonomous-loops SOTA note](../../research/notes/2026-10-23-autonomous-loops-sota.md).
The format and re-injection were surveyed on 2026-10-18 for the first slice
(itd-2610180912189125, its `## SOTA`) and the central view on 2026-10-17; this
pass covers crash safety, the wake, the state vocabulary and the away report.

- **Crash safety is file-system dependent, not a sequence.** Pillai and
  colleagues found 60 crash vulnerabilities across eleven widely used systems
  because application protocols "depend on subtle behaviors of the underlying
  file system" that differ across file systems (OSDI 2014). The practitioner
  summary: POSIX's rename atomicity "only applies to normal operation, not to
  crashes", the parent directory needs its own fsync, and macOS needs
  `F_FULLFSYNC` to reach the disk (Dan Luu). The decision's sequence is the
  right floor; the lab the decision names is what confirms it.
- **The host's stop refusal has a cap and a blind spot.** A Stop hook's block
  continues the turn at most eight times in a row, reset by each tool call
  and adjustable with `CLAUDE_CODE_STOP_HOOK_BLOCK_CAP`;
  `StopFailure` fires instead of Stop on an API error (rate limit, overload,
  authentication) and its output is ignored; the Stop input lists pending
  `session_crons` and `background_tasks` (Claude Code hooks).
- **The host's wakes die with the process.** Session-scoped tasks fire only
  while the session runs and is idle, run late by up to half their gap, expire
  after seven days, and a self-paced loop whose wake comes due while the
  process is down does not fire (Claude Code scheduled tasks). A wake written
  to the checklist as a time is the record; the host's cron is only the device.
- **A state vocabulary to key on, many to one.** A2A's task states
  (`working`; `input-required` and `auth-required` as interrupted,
  non-terminal states; `completed`, `failed`, `canceled`, `rejected` as
  terminal) can be mapped onto the record's decided states: working to
  `working`, waiting to `input-required` or `auth-required` (though abcd's
  waiting also covers waiting on something outside, not only a person), and
  stopped to nothing, since abcd's stopped is not terminal; the mapping is
  lossy, and whether to rename is the first question below; the host's
  `claude agents --json` reports `working`, `blocked`, `done`, `failed`,
  `stopped` and is the supported read, while its state file "is not a stable
  interface" (A2A 1.0; Claude Code agent view).
- **The pre-run sweep and the morning lists have precedents.** The host's
  cloud projects advise sending "one small piece of the real work" before a
  batch and group threads as "Waiting on you"; Temporal's pause (pre-release:
  invite-only on Temporal Cloud, Server v1.30.0 behind a flag self-hosted)
  records identity, reason and request id; GitHub's deployment gate holds a
  job in "Waiting" until a run is cancelled at the 35-day workflow-run limit,
  which includes "time spent on waiting and approval" (the 30-day failure is
  the Enterprise Server 3.5 page's; the current cloud pages do not state a
  limit) (host, Temporal and GitHub docs).
- **The phone can be reached by standards.** Web Push stores a message for a
  `TTL` with a 4096-byte payload floor (RFC 8030), and iOS 16.4 delivers it to
  home-screen web apps without a developer account (WebKit blog): the push
  carries the stop's id, the page fetches the options.

**Path 2**: the checklist as decided (JSON, temp-fsync-rename-dir-fsync,
last-good copy, validate on read), written by the canonical `fsutil` writer
with an `F_FULLFSYNC` option on macOS and proved by the kill-and-reboot lab;
a proposal, against the recorded state decision (working; waiting; stopped),
to map the checklist's states onto A2A's where they fit (working to working,
waiting to input-required or auth-required, stopped with no A2A equivalent)
so the checklist line, the central-view line and a later bridge share words,
the renaming itself being the first question at the end of this section; the host's `claude agents --json` as the
instant local tier where it exists, never its state file; a wake recorded as a
time in the checklist with the host's cron as the device. The Web Push sender
is a Go dependency in the relay, not the binary, and needs explicit sign-off;
a relay-notified phone sits against the local-only stance on agent messaging,
and the plan-time fit-challenge must say where that stance ends. The
independent fit-challenge runs at plan time.

Questions this pass raised for the fit-challenge before the build. They are kept here, not under Open Questions, so they do not hold a build the record already rules ready:

- Whether the checklist's state names adopt A2A's task-state
  vocabulary where it fits (working; input-required or auth-required for
  waiting; no A2A state for stopped) so the central view and a later host or
  protocol bridge share one set of words, which revisits the recorded state
  decision, or keep the decided names and map at the bridge.
- Whether the canonical writer gains an `F_FULLFSYNC` option
  on macOS for the checklist and the away record, and whether the
  kill-mid-write lab also pulls power rather than only killing the process.

## Open Questions

_All resolved at the product thinker's walk and sign-off of 2026-10-23 (dcn-2610230805325396)._

- **Delivery in two specs: settled.** The checklist on disk, its crash-safe write and its re-injection come first, resolving iss-2610222026223032; the waits, trigger promises, timed wakes, the blocker sweep, the return lists, the facilitator's mode and the design-done answer follow. abcd mints one spec per plan, so the second half is carried as the first spec's remainder and split off with `spec close --remainder` when the first lands.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: autonomous sessions stop stalling and come back from a compaction with their work intact once what they own, what each item waits on, their promises and their next look live in a file re-injected at every start and compaction; shown wrong if, over a measured run across at least one night and three automatic compactions, the idle hours while holding doable work and the items lost or repeated after a compaction are no lower than with only the stall fix. Occasioned by the night of 2026-10-16, about ten hours idle in one session with promises lost to compaction, and by iss-2610222026223032.
