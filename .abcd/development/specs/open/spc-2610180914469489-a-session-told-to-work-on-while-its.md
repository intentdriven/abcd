---
id: spc-2610180914469489
slug: a-session-told-to-work-on-while-its
intent: itd-2610180912189125
origin: researcher-authored
production_mode: hand-written
---
# A session left to work on never stops while it has work

## Summary

Delivers itd-2610180912189125, the first slice of the checklist draft
(itd-2610170638184486) and the fix for iss-2610170619247185. Trimmed after
review round 2 to what closes the issue (dcn-2610181017056307). Three parts:

- **The away record.** `abcd mode autonomous [--until <time>]`, `abcd mode
  back` and `abcd mode queue "<step>"` keep one away record per repository on
  the machine, changed only under a lock and lapsing at its end, seven days
  at most. The session-start hook (every source, `compact` included) and the
  prompt hook repeat it to the agent while it holds, and the prompt hook
  shows the person one line saying so, naming `abcd mode back`. The mode
  grants no rights: the person's own standing instruction stays the
  authority to commit, push and open or merge a pull request
  (dcn-2610181016510153).
- **The stop check.** A plugin Stop hook runs `abcd hook stop`. While the
  record holds, a session is only working or waiting: a turn's end is blocked
  while owed work exists, naming the next item, and is allowed only with a
  wake that fires, the host's lateness included, within half an hour while
  work is owed and before the session's reservations need renewing. The host's
  cap of eight consecutive continuations is the outer bound. A StopFailure
  hook records a turn the host ends on an API error. While the person is
  present the check reads disk only and points the work out.
- **The question reroute.** While the record holds, the guard refuses every
  question on the host's question tool. An act only the person may approve
  goes to the queue; every other question is decided and recorded with `abcd
  decide`. `abcd mode back` prints the queue and the recorded stops, one line
  each.

Every behaviour is built test-first: each step names its tests, written and
watched failing before its code. The last step is a live Claude Code session
capture, because the hook mechanics are the host's and a unit test cannot
prove them (principle evaluate-at-the-user-surface).

Moved out by review round 1 (dcn-2610180937181929) and by the trim after
round 2 (dcn-2610181016510153, dcn-2610181017056307), back to the checklist
draft as named remaining work: the report shown once on the person's return,
the status line, the bare `abcd mode` print, making the away mode the
person's act alone (the guard entries, their other routes and the refused
attempts), the guard's reading of `go run ./cmd/abcd` and built binaries as
`abcd`, and the forge listing's reuse cache. "Trim after round 2" below
lists each.

## Design

### The host mechanics this rests on

Read on 2026-10-18 at https://code.claude.com/docs/en/hooks (the old
docs.claude.com hooks page redirects there):

- **Stop.** The hook "runs when the main Claude Code agent has finished
  responding" and not on a user interrupt. Its input carries
  `stop_hook_active`, "true when Claude Code is already continuing as a result
  of a stop hook", and `last_assistant_message`, the text of Claude's final
  response. It also carries `background_tasks` (each with `id`, `type` such as
  `shell`, `subagent`, `monitor` or `workflow`, and `status`) and
  `session_crons` (each with `id`, `schedule` as a cron expression such as
  `0 9 * * 1-5`, `recurring` and `prompt`); both are present only when the
  host's task registry is reachable. Output `{"decision": "block", "reason":
  …}` prevents the stop. Claude Code caps this at eight consecutive
  continuations, overriding the next block; the count resets each time Claude
  calls a tool, and `CLAUDE_CODE_STOP_HOOK_BLOCK_CAP` raises it.
- **StopFailure.** Fires instead of Stop "when the turn ends due to an API
  error"; its matcher is the error type (`rate_limit`, `overloaded`,
  `server_error` and others). It cannot block, and its output is ignored.
- **SessionStart.** The matcher values include `compact` ("auto or manual
  compaction"). Plain stdout, or `hookSpecificOutput.additionalContext`, is
  added to Claude's context.
- **UserPromptSubmit.** Plain stdout or `additionalContext` is added beside
  the prompt, as a system reminder.
- **Universal output.** `systemMessage` is a "warning message shown to the
  user".
- **Where a prompt comes from.** The prompt hook also fires on turns the
  host starts on its own and on a message another session sends, and its
  input has no field naming the prompt's origin, so a hook cannot tell the
  person's word from a relayed one.

Read on 2026-10-18 at https://code.claude.com/docs/en/scheduled-tasks:

- **Scheduled tasks.** "All times are interpreted in your local timezone",
  not UTC. A recurring task "starts each run late by up to half the gap to
  the next run, and by at most 30 minutes", except `*/5 * * * *`, which runs
  15 seconds sooner; a one-shot task set for `:00` or `:30` runs up to 90
  seconds early, and one set for any other minute is not shifted. Day of
  week takes `0` or `7` for Sunday, and when both day fields are restricted
  either matches, "standard vixie-cron semantics". `CronList` rows show each
  task's next run.

abcd's hook manifest already registers SessionStart with no matcher, so
compaction already runs `abcd hook session-start`. It has no Stop or
StopFailure entry, and its PreToolUse entry matches the shell tool and the
host's question tool.

### The away record

- **Where it lives.** One JSON file per repository on the machine, in the
  away store, `~/.abcd.noindex/away/<root-sha>/`, keyed on the repository's
  root commit as the worktree store is (dcn-2610180913399499), with a lock
  file beside it. The same folder holds each working copy's list of branches
  it created (dcn-2610181003463367). Everything in it is written through
  fsutil's atomic writer.
- **What it holds.** Who set it: the holder name reservations cache, without
  a network read, falling back to git's `user.name`, and the working copy it
  was set from. It also holds the start, the end, the queue of steps for the
  person, the stops recorded (with work owed, with no next look, or on an
  API error with its type, each at the cap labelled "possibly forced by the
  cap"), and, once ended by word, who ended it and from which working copy.
- **The lock** (dcn-2610180937181736). Every read-modify-write (`mode
  autonomous`, `mode back`, `mode queue`, a recorded stop, a recorded
  branch) holds `fsutil.WithFileLock` on the lock file and re-reads the
  record inside it. The stop check reads the record and lists the forge
  outside the lock, then takes it, re-reads, and records its stop only if the
  record still holds, so a check that began while away never writes away
  back after `abcd mode back`. A lock not had within two seconds: the verbs
  refuse naming it, and the hooks allow the stop and say so on stderr.
- **Ownership and permissions.** The record and the lock file are used only
  when each is a regular file this account owns that no one else can write,
  within 64 KiB: the checks `rules.json` already gets. A record that fails
  them, or does not parse, reads as not away (failing open, with a stderr
  line), and a verb asked to change it refuses naming the file.
- **When it ends.** It holds while now is before its end. It ends with `abcd
  mode back`, or when its time passes. A human prompt never ends it
  (dcn-2610180913399042).
- **The stale-record cap.** An open-ended record ends seven days after it was
  set, and `--until` may name no later time; a later one is refused with
  nothing written. A run of several days, like the standing run of 2026-10-15
  through 2026-10-17, fits; a forgotten record lapses.
- **Seen by the person.** While the record holds, the prompt hook adds a
  one-line `systemMessage` on every prompt: away until when, set by whom and
  from which working copy, and the literal `abcd mode back` as the way to
  end it. The away block tells the agent to run `abcd mode back` only on the
  person's word in its session (dcn-2610181003474765).
- **A lapsed window with anything unread.** While a record whose end has
  passed is not yet closed and holds a queued step or a recorded stop, the
  prompt hook shows the person one line on every prompt counting them and
  naming `abcd mode back`, until that verb closes the record
  (dcn-2610181003474765).
- **Ending it.** `abcd mode back` prints one plain line per queued step and
  per recorded stop (its label included), records who ended the mode and
  from which working copy, and ends the record; on a record whose time has
  passed it prints the same and closes it. Nothing more is printed: the
  report on return moved to the checklist draft.
- **The badge.** The three-word badge in `internal/core/mode` is untouched.
  While the record holds, `abcd mode product-thinker` and `abcd mode
  facilitator` are refused, naming `abcd decide`, since a badge parked on the
  person is a question waiting for them (dcn-2610180913431493).

### Who sets it, and what it grants

**It grants nothing** (dcn-2610181016510153, superseding
dcn-2610181003456444). The person's own standing instruction stays the
authority to commit, push, and open or merge a pull request, exactly as
before the mode existed. The away mode only keeps a session working while
owed work exists and turns its questions into recorded decisions or queued
steps; it is never treated as permission. The away block says so, in those
words.

**Who sets it.** The person, with their own command (at a terminal, or with
`!` in the session), or a session on the person's word in that session, as
dcn-2610180913399042 first said. Nothing checks which: the prompt hook cannot
tell the person's message from one another session relays. Because the mode
grants no rights, a mode set on a relayed word only keeps sessions working
and deciding instead of asking, and the person sees who set it, from which
working copy, and `abcd mode back` on their next prompt.

**Why the guard entries left the slice.** Round 2's choice (a) made setting
the mode the person's act alone, because the mode then carried push rights.
The OS sandbox of spc-2610161751056096 fences only the agents abcd's runner
launches, never the person's own interactive sessions, which are the ones
that stalled. There, the guard entries would only ever be a mistake filter
(adr-42), so their two days would buy a trust claim the plan cannot make.
With no push rights in the mode there is nothing for them to protect in this
slice. Making the mode the person's act alone (the guard entry, its other
routes, the store's shell-write and file-tool refusals, the refused
attempts, and the capture's checks of the `!` path), and the guard reading
`go run ./cmd/abcd` and built binaries as `abcd` for every entry, the ledger
flip included, are named remaining work of itd-2610170638184486; the second
is a security-relevant widening that gets its own review there.

### Owed work

Owed work is four kinds (dcn-2610180913416683, refined by
dcn-2610180936568358, dcn-2610180936571036 and, in review round 2,
dcn-2610181003463367, dcn-2610181003467233, dcn-2610181003464379 and
dcn-2610181003488355):

1. a live reservation under this copy's key whose renewal is due, or whose
   work no open pull request carries yet;
2. an open pull request of the session's own that is behind its base or
   failing its checks;
3. a branch of its own with commits not pushed;
4. a branch of its own stacked on a change that is still open, while it
   needs work (the clearing rule below).

- **What "own" means.** A branch is the session's own when one of these
  holds (dcn-2610181003463367):
  - this working copy's HEAD reflog has checked it out;
  - the session created it: when the guard's pre-tool hook sees this session
    run `git worktree add -b|-B <b>`, `git switch -c|-C <b>` or `git checkout
    -b|-B <b>`, and no branch named `<b>` exists yet, it records `<b>` under
    this copy's key in the away store, under the store's lock, whether or not
    the person is away. The hook only records; it refuses nothing, and a
    failed write leaves the command allowed with a stderr line. This is how
    the stacked drain branches of 2026-10-17 (`feat/drain-step11` and its
    siblings) are seen: cut with a plain `git worktree add`, they had no
    upstream, no entry in this copy's reflog, no run state and no record id
    in their names. A record made for a command that then failed names a
    branch that does not exist, and is ignored;
  - it is the branch of a lane in a run state this copy's local tier holds
    (`.abcd/.work.local/run/<run-id>/state.json`), while that run's
    reservation is live under this copy's key;
  - it is the head branch of an open pull request linked to a reservation
    under this copy's key (the link below).
- **Live, not stale.** A branch owned through the reflog or a recorded
  creation counts only while it holds a commit authored by this checkout's
  `user.email` within the last 14 days, or a live reservation under this
  copy's key links it. A finished or abandoned run's lanes never count. So
  the five lanes of 2026-10-12 this checkout still holds, from runs that
  stopped at implement and never landed, are not owed forever, and the
  nothing-owed row stays reachable.
- **Never own.** The default branch; a tag; a branch whose open pull request
  another person authored (the listing's author is not the person); and a
  branch this copy only checked out to review, which the authorship rule
  above leaves out, since it holds none of this checkout's commits. A
  sibling copy's branch is none of the routes above, so it never holds this
  session.
- **The reservation filter and link** (dcn-2610181003467233). A reservation
  counts only when its copy key is this copy's (the view's `Copy ==
  CopyThis`). A pull request carries a reservation's work when the record id
  appears in its head branch's name or on a `Resolves:` or `Delivers:` line
  of its body, matched on word boundaries, so iss-5 never matches iss-50. A
  `Refs:` line, which the issue-resolution gate (RS004) requires of any pull
  request that only names an issue, carries the work only when the head
  branch is already own by another route, and never makes a branch own: a
  sibling copy's pull request that mentions the record neither leaves this
  session waiting nor becomes its branch. A renewal is due halfway between
  the reservation's latest claim or renewal and its expiry.
- **Behind** (dcn-2610181003464379). While away, a pull request is behind
  when the listing's merge state (`mergeStateStatus`) is `BEHIND`, which the
  forge measures against its base as it stands now, not against the
  last-fetched default branch, which nothing refreshes overnight. Only a
  branch with no pull request is compared on disk, with the last-fetched
  default branch, and the reason names when that ref was last fetched.
- **Finding a stacked branch.** An own branch is stacked when its open pull
  request's base is not the default branch, or, with no pull request, when
  the nearest other own branch whose tip is an ancestor of its tip is not
  contained in the last-fetched default branch.
- **The clearing rule for a stacked branch** (dcn-2610180936571036, refined
  by dcn-2610181003488355). One whose base is still open is owed only while
  any of these holds: it is behind its base (its merge state, or with no
  pull request its base's tip on disk); its checks are failing; a review of
  its head is outstanding (the listing's review decision is
  `CHANGES_REQUESTED`, or for a run's lane its run state records no passing
  review of its head), since the coordination prep's decision that a
  declared dependency constrains only the landing says its review runs now;
  or it is not verified. Verified means a preflight receipt naming its head
  in any worktree (`landing.Receipt`) where the repository mints receipts
  (a receipt folder exists in one of its worktrees); in a repository without
  them, its pull request's checks passing in the listing; with no pull
  request there either, it is judged on the other conditions alone and the
  reason says it was not verified. Otherwise it is waiting on its base,
  which is not owed: only its landing waits.
- **Unpushed.** An own branch with commits not on its upstream, or, with no
  upstream configured, not in any remote-tracking ref. A branch whose
  upstream is gone (merged and deleted on the remote, a squash merge
  included) never counts.
- **Handed back by the queue** (dcn-2610181016510153). The mode grants no
  push, so an item whose next act is a push (an unpushed branch, a
  reservation whose work no pull request carries, a behind or failing pull
  request once its local fix is committed) may be one the session has no
  standing instruction to send. A queued step that names the item's branch
  or record id, on the same word-boundary match as the link, hands that item
  back to the person for the rest of the window, so the stop check stops
  naming it. A branch is handed back only while it holds commits its upstream
  lacks, so the local work (the merge from the base, the fix, the review, the
  preflight) is done first and only the push waits.
- **What is read, and when (invariant 7,
  adr-2610181003541997).** While the person is present the check reads disk
  only: renewals due (reservations as last fetched, `claim.LastFetched`) and
  unpushed own branches. It never calls the forge, so no turn's end touches
  the network. While away it adds at most one `gh pr list` of the person's
  open pull requests (head and base branch, author, body, merge state,
  review decision, check rollup; each field among those
  `internal/ghcontract` pins) under a ten-second deadline. On a failure or
  timeout it judges from the disk kinds alone, naming the pull requests as
  not read. There is no cache: a run of eight blocks costs up to eight
  listings, each bounded by the deadline (the reuse cache is remaining work
  of the checklist draft). Brief invariant 7 gains this listing as the away
  mode's documented meaning, as the drain's listing is, by
  adr-2610181003541997.
- **Order.** The next item is the first of: a renewal due, a failing pull
  request, a behind one, a stacked branch, an unpushed branch, unhanded work.

### The stop verdict

`autonomy.Judge(record, owed, payload, session, now) Verdict` is pure.

**The allowed states while away** (dcn-2610180936577720): working (the stop
is blocked) or waiting (the stop is allowed with a qualifying wake, whose
next fire time is the next look). abcd never allows a stop with work owed
and no qualifying wake; "stopped" happens only when the host's cap overrides
a block, and is then recorded. Stopped is allowed, and flagged, only while
the person is present.

**The deadline** (dcn-2610181003455731, refining dcn-2610180936579498). The
earlier of: the earliest renewal due among the live reservations under this
copy's key, minus `lease.Skew`; and now plus a ceiling, 30 minutes while work
is owed and one hour when none is. The ceiling closes the long-lease hole: a
release reservation's 24-hour lease puts its renewal 12 hours on, and a
`0 7 * * *` cron seen at 21:00 would otherwise qualify and leave the session
idle for ten hours with work owed.

**The qualifying wake.** A `session_crons` entry qualifies when its latest
fire time is no later than the deadline. The latest fire time is the next
time its schedule matches after now, read in the machine's local time as the
host reads it, plus the most the host may run it late: for a recurring task,
half the gap from that match to the following one, at most 30 minutes, and
nothing for exactly `*/5 * * * *`; a one-shot task is taken as matched, since
the host only runs one early. So an hourly cron whose next match is 3 minutes
before the deadline does not qualify (it may run 30 minutes late), and with
owed work the longest a session can wait is the 30-minute ceiling.

The next fire time comes from a five-field cron parser of abcd's own (minute,
hour, day of month, month, day of week; lists, ranges, steps, `a-b/n` and
`*`; 7 is Sunday as 0 is). It follows vixie cron, as the host says it does:
when both day fields are restricted either matches, and a day field written
from `*` (`*/n` included) counts as unrestricted for that rule. It steps
minute by minute in absolute time, converting each minute to local wall-clock
time, from now to the deadline plus an hour (for the following match), so at
most a few hundred steps; a wall-clock minute skipped by a daylight-saving
change never matches, and one repeated by it matches at its first
occurrence. An expression it cannot parse never qualifies. A background task
never counts on its own while away. No new dependency.

**The session's state.** One file per session id in the local tier
(`.abcd/.work.local/autonomy/sessions/<session-id>.json`; an id outside
`[A-Za-z0-9_-]{1,128}` is refused and the stop allowed with a stderr line)
holds the state, the next look, the owed-set fingerprint, the last question
message's hash, and the count of consecutive blocks.

While away, the rows are read in order and the first that matches decides:

| # | condition | verdict |
|---|---|---|
| 1 | `last_assistant_message` ends on a line ending in `?`, not yet blocked for this message | block with the reroute's reason (working) |
| 2 | owed, set changed or first block this session | block: next item and count (working) |
| 3 | owed, set unchanged, qualifying wake | allow (waiting; next look kept) |
| 4 | owed, set unchanged, no qualifying wake | block: next item, and if it cannot move, take another or schedule a look before the deadline (working) |
| 5 | none owed, qualifying wake | allow (waiting; next look kept) |
| 6 | none owed, no qualifying wake | block: take the next ready item first (reserve it), and only if none can be taken alone, schedule a look before the deadline (working) |

- **An item that needs a push** (dcn-2610181016510153). The reason for it
  says: do the local work, then push if your standing instruction allows;
  otherwise queue the push with `abcd mode queue "<step>"` naming the branch
  or record id, and take the next item. It never says the away mode allows
  the push.
- **`stop_hook_active`** decides no row, so both values are covered by every
  row. It alone drives the count (dcn-2610181003484631, refining
  dcn-2610180936577720): a block with it `false` sets the count to one, and
  with it `true` adds one. A change in the owed set no longer resets the
  count, since the host's count does not: in the round-2 scenario a renewal
  falling due at the sixth block would have reset abcd to one, and the host's
  override at its ninth would have found abcd at three, unrecorded. The block
  whose count passes the cap (eight, or `CLAUDE_CODE_STOP_HOOK_BLOCK_CAP`
  when set) records "stopped with work owed" (rows 2 and 4) or "stopped with
  no next look" (row 6) in the away record, labelled "possibly forced by the
  cap", because the host's count also resets on a tool call abcd cannot see.
  The count so errs toward recording a stop that did not happen, never
  toward missing one, and step 7 checks where the host's count resets.
- **Present.** With owed work (the disk kinds only), allow with a
  `systemMessage` naming the work (stopped, flagged); with none, allow
  silently. No forge call.
- **The front door.** `abcd hook stop` maps the payload and writes the host's
  JSON. A malformed payload, an unreadable record or a lock not had allows
  the stop and says so on stderr, following the guard's fail-open-loud
  contract. A stop is never blocked on abcd's own fault.
- **StopFailure** (dcn-2610180937175629). `abcd hook stop-failure`, while
  the record holds, records "stopped on an API error" with the error type,
  under the lock. It cannot continue the session; the scope conditions name
  the gap.

### The question reroute

`questionGate` (`internal/surface/cli/guard_question.go`) reads the away
record first. While it holds, it denies every question-tool call, abcd's or
another tool's, with one reason whatever the chip (dcn-2610180937177502).
The classification is by the act asked about, which the reason lists, so the
agent routes it (dcn-2610181003477194, extended by dcn-2610181016510153):

- **Queued for the person** with `abcd mode queue "<step>"`, carrying on with
  other work meanwhile, is a question about an act only the person may
  approve:
  - an approval or ruling the repository binds to the person per act: an
    embargo close's `--approve <digest>`, which `commands/embargo.md` says is
    passed only after the person, in this session, has read the text and
    said yes, and never supplied on the agent's own; the embargo triage
    verdict ("the verdict is the person's decision", `commands/embargo.md`);
    the drain's hand-back ("a hand-back is a person's decision",
    `commands/drain.md`); the confirmed apply verb for remote configuration
    (brief invariant 10); a release sign-off;
  - pushing, opening or merging a pull request without a standing
    instruction of the person's that allows it; the away mode is never that
    instruction;
  - publishing anything, changing a repository's visibility, or posting a
    comment or message outward, other than a pull request a standing
    instruction allows;
  - adding a dependency;
  - a secret, a credential or an App;
  - a step only a website or a browser can do;
  - a destructive act: deleting what the session did not create, overwriting
    uncommitted work, a force push, discarding a peer's work;
  - anything that spends money;
  - any other act a command page or a brief invariant names as the person's.
- **Decided and recorded** with `abcd decide`, with its grounds and how to
  reverse it, is every other question.

**A release sign-off** is a cut, or a step of one, that asks for the person
by name, or that no standing instruction of theirs allows. A standing
permission for automatic cuts stands; the away mode adds none.

**Nothing detects the act mechanically.** The agent classifies its own
question from the refusal's list, and the guard's existing entries enforce
only what they name, a force push and a skipped hook among them; the rest of
the list rests on the agent following it, as the person-only rules on the
command pages already do.

The away block repeats this list and that the mode grants nothing. No mode
is set and no question is marked open.

## Footprint

- packages: internal/core/autonomy, internal/core/mode, internal/core/guard, internal/surface/cli, hooks, commands/mode.md, commands/guard.md, .abcd/development/brief/02-constraints/03-invariants.md, .abcd/development/brief/04-surfaces/08-abcd.md, .abcd/development/brief/04-surfaces/17-guard.md
- tests: internal/core/autonomy, internal/core/mode, internal/core/guard, internal/surface/cli

## Steps

1. The away record, its store, its lock and its verbs
   - packages: internal/core/autonomy, internal/core/mode, internal/surface/cli, commands/mode.md, .abcd/development/brief/04-surfaces/08-abcd.md
   - tests: internal/core/autonomy, internal/core/mode, internal/surface/cli
   - Watched failing first: TestAwayRecordSharedAcrossWorktrees (`abcd mode
     autonomous --until 09:00` writes one record in the away store that a
     sibling worktree reads, with who set it and from which copy);
     TestUntilClockTimeAndTimestamp;
     TestUntilRefusedPastOrNoTierOrNoCheckout (nothing written);
     TestStaleRecordCapSevenDays (an open-ended record reads as ended seven
     days on, and `--until` beyond that is refused);
     TestModeBackPrintsQueueAndStops (one plain line per queued step and per
     recorded stop, the "possibly forced by the cap" label included, and the
     ender and copy recorded; also on a record whose time has passed);
     TestRecordPastEndReadsEnded; TestQueueAppends;
     TestConcurrentWritesUnderLock (two `mode queue` calls racing keep both
     steps, and a stop recorded after `mode back` re-reads inside the lock
     and does not turn the mode back on); TestRecordOwnershipFailsOpen (a
     record or lock that is a symlink, not this account's, writable by
     others, or over 64 KiB reads as not away, and a verb refuses naming
     it); TestBadgeRefusedWhileAway; the badge's three-word vocabulary test
     passes unchanged. The plugin page and the brief's `abcd mode` surface
     text describe the three verbs and say the mode grants no rights, the
     person's standing instruction deciding what is pushed, and `go generate
     ./internal/surface/cli` refreshes the derived appendix.
2. Repeated to the agent, shown to the person
   - packages: internal/surface/cli
   - tests: internal/surface/cli
   - Watched failing first: TestSessionStartAwayBlockEverySource (`startup`,
     `resume` and `compact` print the away block while the record holds,
     nothing when it does not, and the block says the mode grants nothing,
     carries the person-only act list and the rule to run `abcd mode back`
     only on the person's word); TestPromptRouterAwayBlockEveryPrompt (a
     second prompt in the same session too, outside the loader's dedup
     ledger); TestPromptRouterAwaySystemMessage (one line the person sees on
     every prompt while it holds, naming who set it, from which copy, and
     the literal `abcd mode back`); TestPromptRouterLapsedWindowLine (a
     lapsed, unclosed record with a queued step or a recorded stop shows one
     line naming `abcd mode back` on every prompt until `mode back` closes
     it, and one with neither shows nothing); all inside the hook's existing
     deadline.
3. The created-branch recorder
   - packages: internal/core/guard, internal/core/autonomy, internal/surface/cli, commands/guard.md, .abcd/development/brief/04-surfaces/17-guard.md
   - tests: internal/core/guard, internal/core/autonomy, internal/surface/cli
   - Watched failing first: TestBranchRecorderOnCreate (`git worktree add -b
     feat/x ../x`, `git switch -c feat/y` and `git checkout -B feat/z`
     record the branch under this copy's key, whether or not the person is
     away, only when no branch of that name exists, and `git checkout -b
     main` on an existing `main` records nothing); TestBranchRecorderNeverRefuses
     (the command is allowed whatever the recorder does, and a store it cannot
     write leaves a stderr line); TestBranchRecorderUnderLock (two recordings
     racing keep both). commands/guard.md and the brief's guard chapter say
     the hook records a created branch and refuses nothing for it.
4. Owed work, and invariant 7's clause
   - packages: internal/core/autonomy, .abcd/development/brief/02-constraints/03-invariants.md
   - tests: internal/core/autonomy
   - Watched failing first, against real git repositories in temp folders
     and a fake forge listing pinned to `internal/ghcontract`'s fields:
     TestOwnBranchFromReflogWithoutWindow (checked out before the mode was
     set, with a recent commit of this checkout's, still own);
     TestOwnBranchFromRecordedCreation (a branch cut with `git worktree add
     -b`, absent from this copy's reflog, with no upstream and no record id,
     is own once recorded); TestOwnBranchFromLiveRunLane (a lane of a run
     whose reservation is live under this copy's key is own; a lane of a run
     whose reservation lapsed, or of a finished run, is not);
     TestOwnBranchFromLinkedReservation; TestStaleOwnBranchNotOwed (a
     reflog or recorded branch whose newest commit of this checkout's is
     older than 14 days, with no live reservation, is not owed);
     TestReviewCheckoutNotOwn (a branch checked out to review, holding only
     another author's commits, and the head of a pull request another person
     authored, are never own); TestSiblingBranchNotOwn;
     TestReservationCopyKeyFilter (a sibling copy's live reservation is not
     owed); TestReservationLinkByHeadOrClosingTrailer (the record id in the
     head branch or a `Resolves:`/`Delivers:` line links; a green linked
     pull request is not owed until renewal is due);
     TestRecordIDWordBoundary (iss-5 does not match iss-50 in a branch name
     or a trailer); TestRefsLineCarriesOnlyOwnBranch (a `Refs:` line on a
     sibling copy's pull request neither carries the work nor makes its
     branch own; on an own branch's pull request it carries it);
     TestBehindFromMergeState (a pull request whose merge state is `BEHIND`
     is owed while the local default branch is stale and shows it current;
     one `CLEAN` is not); TestStackedBranchFoundWithoutPullRequest;
     TestStackedBranchClearingRule (owed while behind its base, failing,
     with a review outstanding, or unverified; waiting on its base
     otherwise); TestClearingRuleReview (`CHANGES_REQUESTED` in the listing,
     and a run lane with no passing review of its head, are owed);
     TestClearingRuleWithoutReceipts (in a repository with no receipt
     folder, passing checks verify; with no pull request the reason says it
     was not verified); TestUnpushedSkipsGoneUpstream (a squash-merged
     branch whose upstream is gone is not owed); TestQueuedPushHandsItemBack
     (a queued step naming an unpushed branch hands it back while the branch
     is ahead of its upstream; one naming a behind branch with nothing to
     push does not; iss-5 queued does not hand back iss-50);
     TestPresentReadsDiskOnly (zero forge calls while present);
     TestForgeListingDeadline (a hung listing ends at ten seconds, and the
     check judges from the disk kinds with a "not read" note);
     TestOwedOrderFixed. Brief invariant 7 gains the away mode's one listing
     per check and its deadline, citing adr-2610181003541997.
5. The stop verdict, the Stop and StopFailure hooks
   - packages: internal/core/autonomy, internal/surface/cli, hooks
   - tests: internal/core/autonomy, internal/surface/cli
   - Watched failing first: TestJudgeRows (each of the six away rows and both
     present rows, each with `stop_hook_active` true and false);
     TestJudgeEscapeBoundReproducesNoStall (owed, unchanged, `true`, no wake:
     blocked, not allowed); TestJudgeCapRecordsStop (the block passing the
     cap, and an override from the environment, records the stop labelled
     "possibly forced by the cap"); TestCountResetsOnlyOnInactive (a change
     in the owed set at the sixth block does not reset the count, and the
     ninth consecutive block still records the stop; a block with
     `stop_hook_active` false sets it to one); TestPushItemReasonDefersToStandingInstruction
     (the reason for an unpushed branch says to push only if the standing
     instruction allows, otherwise to queue it, and never names the away mode
     as permission); TestDeadlineCeiling (with
     work owed and no reservation, now plus 30 minutes; with nothing owed,
     now plus one hour; with a renewal due sooner, that minus the skew);
     TestCronNextFire (lists, ranges, steps, both day fields, an unparsable
     expression never qualifies); TestCronDayOfWeekSeven (7 matches Sunday
     as 0 does); TestCronRangeStep (`10-50/20` matches minutes 10, 30 and
     50); TestCronStarStepInDayField (`0 9 */2 * 1` matches only days that
     are both odd-numbered and Monday, since a field written from `*` counts
     as unrestricted, while `0 9 1 * 1` matches either);
     TestCronDaylightSaving (in Europe/London, `30 1 * * *` has no match on
     the spring-forward day and matches once, at its first occurrence, on
     the fall-back day); TestQualifyingWakeDistantCron (a release
     reservation with a 24-hour lease claimed at 20:00, and `0 7 * * *` seen
     at 21:00: does not qualify); TestQualifyingWakeJitter (an hourly cron
     whose next match is 3 minutes before the deadline does not qualify;
     `*/5 * * * *` and `*/10 * * * *` do; a one-shot at the deadline does);
     TestQualifyingWakeRecurringCron (`*/30 * * * *` with nothing owed
     does); TestQualifyingWakeAfterLeaseLapse (a wake past the earliest
     renewal due minus the skew does not); TestBackgroundTaskNotAWake (a
     `shell` task running `sleep 36000` or a `tail -f` does not count);
     TestNextLookStored; TestProseQuestionBlockedOnce; TestNothingOwedReason
     (names taking the next ready item first); TestSessionIDShape;
     TestHookStopOutput (`{"decision":"block","reason":…}` only on a block, a
     `systemMessage` while present; a malformed payload, an unreadable record
     or a held lock allows with a stderr line); TestHookStopFailureRecords
     (the error type recorded while away, nothing written while present);
     `hooks/hooks.json` gains Stop and StopFailure entries, and `hooks/stop.sh`
     and `hooks/stop-failure.sh` take the same bootstrap and home checks as
     `hooks/subagent-stop.sh` (the hook-shape tests cover them); `go generate
     ./internal/surface/cli` refreshes the derived appendix.
6. The question reroute
   - packages: internal/surface/cli, internal/core/autonomy, .abcd/development/brief/04-surfaces/17-guard.md
   - tests: internal/surface/cli, internal/core/autonomy
   - Watched failing first: TestAwayQuestionDeniedOneReason (a Product-chip,
     a Tech-chip, a Setup-chip and an unchipped question, and another tool's,
     get the same reason); TestAwayReasonListsPersonOnlyActs (the reason
     names every act in the list: the embargo `--approve` digest, the
     embargo triage verdict, the drain's hand-back, the remote-configuration
     apply verb, a release sign-off and its meaning, pushing, opening or
     merging a pull request without a standing instruction, an outward
     comment or message, and the catch-all; that the away mode is no
     standing instruction; and the route to `abcd mode queue` for those and
     to `abcd decide` for the rest); TestAwayQuestionSetsNoMode (no mode
     set, no marker written); TestPresentQuestionGateUnchanged. The brief's
     guard chapter says what the gate does while away, and that nothing
     detects the act mechanically.
7. The live session capture
   - packages: none (evidence only, kept in the local tier)
   - tests: a live Claude Code session with the plugin built from the branch
   - In a lab checkout, seed one behind pull request checked out before the
     mode is set, one stacked branch made with `git worktree add -b` in the
     session, and one unpushed branch, and record the transcript. It must
     show: the mode set and the away block on start, with the person's line
     naming who set it and `abcd mode back`; a manual `/compact` followed by
     the block again; an attempted stop blocked naming the behind pull
     request; a reply with no tool call blocked again rather than allowed;
     the stacked branch's item; the unpushed branch's reason deferring to the
     standing instruction, and, with none given, the push queued and the
     item no longer named; a cron set for the next morning refused as a wake
     and one inside the deadline accepted, with abcd's computed fire time
     compared with the `next run` the host's `CronList` row shows; nine
     blocks in a row with no tool call, then the same with one tool call in
     between, to see where the host's count resets; a question refused with
     the one reason; the window left to lapse with a queued step, and the
     next prompt's line naming `abcd mode back`; and `abcd mode back`
     printing the queue and the recorded stops. Repeat with the mode off to
     show the stop allowed and flagged with no forge call. Capture before and
     after the change on the same checkout, as the release lesson on host
     fixes requires.

## Size

Re-estimated after the trim at 5.75 working days, down from eight
(dcn-2610181017056307):

| step | days | was |
|---|---|---|
| 1. The away record, its store, its lock and its verbs | 0.75 | 1.0 |
| 2. Repeated to the agent, shown to the person | 0.5 | 0.75 |
| 3. The created-branch recorder | 0.25 | (in the old step 2, 1.5) |
| 4. Owed work, and invariant 7's clause | 1.75 | 2.0 |
| 5. The stop verdict, the Stop and StopFailure hooks | 1.5 | 1.5 |
| 6. The question reroute | 0.5 | 0.5 |
| 7. The live session capture | 0.5 | 0.75 |
| total | 5.75 | 8.0 |

Step 1 loses the refused attempts and the fuller `mode back` print; step 2
the attempt line; step 4 the listing's reuse cache and its stale-cache
fallback, less the small queue hand-back it gains; step 7 the guard, `!`
and relay checks. The old step 2's guard entries leave whole, and only its
branch recorder stays, as step 3.

**Why not four.** About four working days is not reachable by scoping
alone. Steps 3 to 5 carry 3.5 days, and every rule in them answers a
review finding the trim keeps: the created-branch, stale-ownership and
`Refs:` rules (round-2 blockers 3 to 5), the merge state (blocker 2), and the
deadline and cron cases (blocker 1). Cutting any of them reopens a blocker,
and the stacked-branch rules are the issue's second instance. The floor is
about 5.75 days of work. The elapsed time can be shorter: `Judge` is pure
over the owed set, so steps 3 and 4 can run in one lane while steps 1, 2, 5
and 6 run in another once step 1 fixes the record's shape, which puts the
critical path at about four days with the capture after both. Whether to run
two lanes is the release orchestrator's choice.

Round 2 estimated eight days for seven steps, round 1 five days for six; the
first plan wrote no size down.

## Review round 1

On 2026-10-18 an Opus and Sonnet pair reviewed the plan as frozen at
82b928305, with the verdict "fix the plan first". Each item and how it was
handled:

| # | finding | handled |
|---|---|---|
| 1 | the ownership rule misses both instances; stacked-branch clearing | fixed: "own" without a window, from the reflog, a run's lanes or a linked reservation; the clearing rule; stacked found without a pull request; gone upstreams skipped (dcn-2610180936568358, dcn-2610180936571036) |
| 2 | the escape bound reproduces the stall | fixed: an unchanged owed set is allowed only with a qualifying wake, else blocked again, the host's cap the outer bound with the stop recorded; `stop_hook_active` decides no row, so the missing row is covered (dcn-2610180936577720) |
| 3 | a wake's timing is never checked | fixed: next fire time from abcd's own cron parser, before the earliest renewal due minus the skew; background tasks never count; distant, recurring and stale-shell rows tested (dcn-2610180936579498) |
| 4 | the reroute approves acts only the person may approve | fixed: classified by the act with the listed person-only acts, the embargo `--approve` digest included; the chip no longer routes (dcn-2610180937177502) |
| 5 | stopping while away contradicts the coordination decision | fixed: while away only working or waiting with a qualifying wake; the next look stored in the session's state (dcn-2610180936577720) |
| 6 | concurrent writes lose updates | fixed: `fsutil.WithFileLock` with a re-read inside it; ownership and permission checks; an unreadable record fails open (dcn-2610180937181736) |
| 7 | a stale record traps the person | fixed in part: capped at seven days rather than 24 hours, since the record's standing run of 2026-10-15 to 2026-10-17 would not fit 24 hours and requiring `--until` contradicts the open-ended ruling; the person sees a line on every prompt rather than past a threshold; the agent runs `abcd mode back` only on the person's word (dcn-2610180937181736) |
| 8 | the return list reads one working copy's decisions | moved: the return list leaves the slice with the split, and the fix (`abcd decide` appending each id to the away record under its lock) moves with it (dcn-2610180937181929) |
| 9 | present mode undefined; per-turn forge call | fixed: present reads disk only with no forge call; away listing under a ten-second deadline with a cache (dcn-2610180936571036) |
| 10 | reservations: no link, no copy filter | fixed: the record id in the head branch or a body trailer links them; only this copy's key counts (dcn-2610180936568358) |
| 11 | StopFailure and prose questions | fixed: a StopFailure entry records the stop and its type, the gap named in the scope conditions; a prose question is blocked once (dcn-2610180937175629) |
| 12 | waiting does not look for other work | fixed: the nothing-owed reason says to take the next ready item first (dcn-2610180936577720) |
| 13 | a refused session can lift away mode for all | fixed in part: `abcd mode back` records who ended it and from which copy, and the away block allows it only on the person's word; showing it on return moved with the return report (dcn-2610180937181736, dcn-2610180937181929) |
| size | no size written; five days more honest | fixed: re-estimated at five days, per step above |
| split | move step 6 and the return half of step 5 | made: the return report, the status line, the bare print and their brief text moved to the checklist draft; kept here, because the slice's own behaviour needs them, are `abcd mode back` printing the queue, invariant 7's clause and the shipped verbs' docs (dcn-2610180937181929) |

## Review round 2

On 2026-10-18 an Opus and Sonnet pair re-reviewed the plan as frozen at
d04bc92fc, with the verdict "fix the plan first". They found round 1's items
2, 5, 6, 11, 12 and 13 fixed, item 7 sufficient, and items 8, 9 and 10
narrowed by the findings below. Each item and how it was handled:

| # | finding | handled |
|---|---|---|
| (a) | away mode as the person's ask to push is not safe: the prompt hook cannot tell a relayed message from the person's, so an agent could grant itself and every copy seven days of push rights | fixed: guard entries refuse every route to setting the mode the guard reads (the binary by any of its names, a `go run` of `cmd/abcd`, `sh -c`, a script file, a shell write into the away store, the file tools), so the person sets it with `!` or at a terminal; refused attempts are recorded and shown; the grant is narrow and its exclusions stated (never the default branch, a tag, a release cut, another person's pull request, a review-only branch, a force push); the guard is called a mistake filter, the OS fence the boundary for launched agents only; the capture checks the `!` path and a relayed `!`, with the no-push fallback named if they break the premise (dcn-2610181003456444) |
| (b) | the cap count can miss a forced stop when the owed set changes | fixed: the count resets only on `stop_hook_active` false, a stop at the cap is labelled "possibly forced by the cap", and the capture checks where the host resets (dcn-2610181003484631) |
| (c) | cron in local time is correct; add a capture check | fixed: the capture compares abcd's computed fire time with the host's listed next run (dcn-2610181003455731) |
| 1 | the wake deadline still allows the stall, through long leases and the host's lateness | fixed: the deadline is the earlier of the renewal due minus the skew and now plus 30 minutes while owed (one hour when not); a recurring cron's fire time adds half its gap, at most 30 minutes; TestQualifyingWakeDistantCron is pinned to a 24-hour lease; daylight saving, day-of-week 7, `a-b/n` and `*/n` in a day field are tested (dcn-2610181003455731) |
| 2 | "behind" is measured against a default branch nothing refreshes overnight | fixed: while away, behind is the listing's merge state; only a branch with no pull request is compared on disk, naming the fetch time (dcn-2610181003464379) |
| 3 | instance 2 is still missed: the stacked drain branches have no upstream, reflog entry, run state or record id | fixed: the pre-tool hook records a branch the session creates (`git worktree add -b`, `git switch -c`, `git checkout -b`) under this copy's key in the away store, only when no branch of that name exists; the capture seeds one (dcn-2610181003463367) |
| 4 | a `Refs:` line is a mention, not a carrier, and iss-5 matches iss-50 | fixed: linked by head branch, `Resolves:` or `Delivers:` on word boundaries; a `Refs:` line carries only for an already-own branch and never confers ownership (dcn-2610181003467233) |
| 5 | stale ownership leaves abandoned lanes and pruned branches owed forever | fixed: a run lane counts only while its run's reservation is live under this copy's key, finished runs' lanes never; a reflog or recorded branch only with a commit of this checkout's in the last 14 days or a live linked reservation (dcn-2610181003463367) |
| 6 | the person-only list is incomplete | fixed: the embargo triage verdict, the drain's hand-back, the remote-configuration apply verb, a push or merge beyond own branches, outward comments and messages, and a catch-all added; release sign-off defined beside the standing permission for automatic cuts; stated that nothing detects the act mechanically and the existing guards enforce only what they name (dcn-2610181003477194) |
| 7 | the queue goes unread when the window lapses | fixed: a lapsed, unclosed record with anything unread shows a line naming `abcd mode back` on every prompt until closed, and the away line carries the literal verb (dcn-2610181003474765) |
| minor | the clearing rule needs abcd's preflight receipt | fixed: where a repository mints no receipts, passing checks verify; with no pull request the reason says unverified (dcn-2610181003488355) |
| minor | the clearing rule leaves out review | fixed: an outstanding review (`CHANGES_REQUESTED`, or a run lane with no passing review of its head) keeps a stacked branch owed (dcn-2610181003488355) |
| minor | the intent still lists a superseded decision | fixed: dcn-2610180913431985 removed from the intent's list; its successor dcn-2610180937181929 stays |
| minor | mint an ADR for the invariant 7 amendment | made: adr-2610181003541997, proposed |
| size | not a finding; re-estimated for the fixes | eight working days, step by step above |

## Trim after round 2

On 2026-10-18 the release orchestrator ruled that the plan take the fallback
and be trimmed to what closes iss-2610170619247185, aiming at about four
working days, because the security and bug-fix cut waits on this slice. The
reviewer added which items stay. Two decisions record it:
dcn-2610181016510153 (the away mode grants no push rights; supersedes
dcn-2610181003456444) and dcn-2610181017056307 (the trim and what moved).

**What stays** is what serves the three things the issue needs: (i) a
session with owed work cannot end its turn silently (the Stop hook, the
ownership rules, the qualifying wake, the cap handling); (ii) questions
reroute by the act (decided with `abcd decide`, or queued when only the
person may approve); (iii) the away record (set, ended, lapsing,
re-injected at start, compaction and prompt, with its lock).

**What moved** to itd-2610170638184486, as named remaining work, and why:

| moved | why |
|---|---|
| the guard entry refusing an agent's `abcd mode autonomous`, and its routes through `sh -c` and script files | the mode grants no rights now, so there is nothing for it to guard in this slice; and the OS fence (spc-2610161751056096) never covers the person's interactive sessions, so it could not become a boundary for the sessions that stalled |
| the guard reading `go run ./cmd/abcd` and built `abcd-<goos>-<arch>` binaries as `abcd`, for every entry, the ledger flip included | a security-relevant widening of what every entry matches, which deserves its own review in the checklist work, not a ride on this cut |
| the shell-write and file-tool refusals on the away store, and the PreToolUse matcher gaining the file tools | they only protected the person-only setting |
| the refused attempts: recorded, shown on the next prompt, listed by `abcd mode back` | they only reported the person-only setting |
| the capture's checks of the person's `!` command and of a relayed `!` | they tested the push-rights premise the fallback removes; the checklist work needs them when it makes the mode person-only |
| the away forge listing's ten-minute reuse cache and its stale-cache fallback | tuning: each away check makes one listing under its ten-second deadline, and a failed read judges from disk, naming the pull requests as not read, so nothing is hidden; a run of eight blocks costs up to eight bounded listings |
| `abcd mode back` printing more than one plain line per queued step and per recorded stop | the report on return is already the checklist draft's; the plain lines keep the queue read |

**Added by the fallback** (dcn-2610181016510153): the reason for an item that
needs a push tells the session to push if its standing instruction allows
and otherwise to queue the push, never treating the away mode as
permission; a queued step that names the item's branch or record id hands it
back for the window, a branch only while it holds commits to push; and the
person-only acts gain pushing, opening or merging a pull request without a
standing instruction. Without the hand-back, a session with no standing
instruction to push would be blocked on its unpushed branch until the host's
cap.

**Each round-2 item after the trim:**

| # | round-2 finding | state after the trim |
|---|---|---|
| (a) | away mode as the person's ask to push is not safe | closed by the fallback the review named: the mode grants no push rights, so a relayed word cannot grant any; the guard entries that answered it the other way moved out with dcn-2610181003456444, which dcn-2610181016510153 supersedes. A mode set on a relayed word only keeps sessions working and deciding, and the person sees who set it on the next prompt |
| (b) | the cap count can miss a forced stop | kept: the count resets only on `stop_hook_active` false, the stop at the cap is labelled "possibly forced by the cap", and the capture checks where the host resets (step 5, step 7) |
| (c) | cron in local time; a capture check | kept: the capture compares abcd's fire time with the host's `CronList` next run (step 7) |
| 1 | the wake deadline still allows the stall | kept whole: the ceiling, the host's lateness, the 24-hour-lease case and the daylight-saving, day-7, `a-b/n` and `*/n` cases (step 5) |
| 2 | "behind" against a stale default branch | kept: the forge's merge state while away, disk only for a branch with no pull request (step 4); the listing loses only its reuse cache, and invariant 7's clause and adr-2610181003541997, amended while proposed, say one listing per check |
| 3 | instance 2 missed: the stacked drain branches | kept, in any mode: the pre-tool hook records a created branch, now its own step 3, and it refuses nothing; the capture seeds one (step 7) |
| 4 | `Refs:` as a carrier; iss-5 matching iss-50 | kept: linked by head branch or a closing trailer on word boundaries, a `Refs:` line carrying only for an already-own branch (step 4) |
| 5 | stale ownership | kept: run lanes only while their reservation is live, reflog and recorded branches only with a commit of this checkout's in 14 days (step 4) |
| 6 | the person-only list is incomplete | kept, and extended by the fallback: pushing, opening or merging a pull request without a standing instruction joins it, replacing "a push or merge beyond the session's own branches" (step 6) |
| 7 | the queue goes unread when the window lapses | kept: the every-prompt line and the lapsed-window line, both naming `abcd mode back`; the lapsed line now counts queued steps and recorded stops, the refused attempts having moved (step 2) |
| minor | the clearing rule needs abcd's receipt | kept: passing checks verify where no receipts are minted (step 4) |
| minor | the clearing rule leaves out review | kept: an outstanding review keeps a stacked branch owed (step 4) |
| minor | the intent lists a superseded decision | unchanged, still fixed; dcn-2610181003456444 is now superseded too and the intent says so |
| minor | mint an ADR for invariant 7 | kept: adr-2610181003541997, proposed, amended to drop the cache from its decision |
| size | re-estimate | 5.75 days, step by step under Size; about four is the elapsed time with two lanes, not the work |

**Round-1 items the trim touches:** item 9 (the per-turn forge call) stays
fixed, since present mode makes no call and the away call is bounded, though
no longer cached; item 13 keeps the recorded ender, while showing it on
return had already moved; items 6, 7 and 11 are kept whole.

**No blocker is reopened.** Nothing the review made essential to (i) to
(iii) was cut. The items kept against the trim's pressure, and why, are
listed in dcn-2610181017056307: each is the fix of a finding above.
