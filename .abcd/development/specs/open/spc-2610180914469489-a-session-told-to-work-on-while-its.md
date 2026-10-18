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
review round 2 to what closes the issue (dcn-2610181017056307), and
narrowed by review round 3 (dcn-2610181035355741, dcn-2610181035361997).
Three parts:

- **The away record.** `abcd mode autonomous [--until <time>]
  [--hold-pushes]`, `abcd mode back` and `abcd mode queue "<step>"` keep one
  away record per repository on
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
  work is owed and before the session's reservations need renewing, and a
  wake releases an unchanged set of work the session can do itself at most
  twice in a row. The host's
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
  calls a tool, and `CLAUDE_CODE_STOP_HOOK_BLOCK_CAP` raises it. Read again on
  2026-10-18 ("Stop decision control"): `hookSpecificOutput.additionalContext`
  is "Non-error feedback for Claude. The conversation continues so Claude can
  act on it, but unlike `decision: "block"` it is shown in the transcript as
  hook feedback rather than a hook error"; use it "when the hook is working as
  designed", and it "keeps the conversation going through the same loop
  protections as `decision: "block"`, namely the `stop_hook_active` input and
  the 8-consecutive-continuation cap, but the transcript labels it `Stop hook
  feedback` and no hook error notification is shown". The stop check blocks
  with `{"hookSpecificOutput":{"hookEventName":"Stop","additionalContext":…}}`
  (Diff review round 1, item 7).
- **StopFailure.** Fires instead of Stop "when the turn ends due to an API
  error"; its matcher is the error type (`rate_limit`, `overloaded`,
  `server_error` and others). It cannot block, and its output is ignored.
  Its input carries `error`, "the error type and is used for matcher
  filtering", with optional `error_details` and `last_assistant_message`
  (here the API error string); "StopFailure hooks have no decision control".
  `abcd hook stop-failure` reads `error`.
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
  was set from. It also holds the start, the end, whether the person said
  pushes wait (`--hold-pushes`), the queue of steps for the person, the items
  the stop check handed back (branch and head), the stops recorded (with work
  owed, with no next look, or on an API error with its type, each at the cap
  labelled "possibly forced by the cap"; a stop identical to the last one
  recorded is kept as that entry with a count and the last time, so a wake
  every ten minutes cannot push the record past its 64 KiB check), and, once
  ended by word, who ended it and from which working copy.
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
  from which working copy, "N steps queued, M items handed back", and the
  literal `abcd mode back` as the way to end it (dcn-2610181035355741). The away block tells the agent to run `abcd mode back` only on the
  person's word in its session (dcn-2610181003474765).
- **A lapsed window with anything unread.** While a record whose end has
  passed is not yet closed and holds a queued step or a recorded stop, the
  prompt hook shows the person one line on every prompt counting them and
  naming `abcd mode back`, until that verb closes the record
  (dcn-2610181003474765).
- **Ending it.** `abcd mode back` prints one plain line per queued step, per
  handed-back item and per recorded stop (its label and count included),
  records who ended the mode and
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

**Pushes held** (dcn-2610181035355741). `--hold-pushes` is the person's word
that pushes wait for their return. It only narrows what sessions do and grants
nothing, so dcn-2610181016510153 stands: like the mode, nothing can check who
ran it, and set on a relayed word it can at worst make sessions queue pushes
they could have sent. It is the only thing that lets a queued push hand an
item back (Owed work, below).

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
    run `git worktree add -b|-B <b>`, `git switch -c|-C <b>`, `git checkout
    -b|-B <b>` or `git branch <b>`, and no branch named `<b>` exists yet, it
    records `<b>` under this copy's key in the away store, under the store's
    lock, whether or not the person is away. A bare `git worktree add <path>`
    records the branch git names after the folder, `git worktree add <path>
    <existing>` records `<existing>`, and `git branch -m|-M <old> <new>`
    carries a record from `<old>` to `<new>`; branch names match as whole
    tokens, and the file is capped (the builder sets the bound, oldest
    dropped). The hook only records; it refuses nothing, and a
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
- **Which copy.** A session's copy is the worktree it started in: the
  session-start hook notes the worktree root of its payload's `cwd` per
  session id in the away store, and the recorder and the stop check read the
  copy key, local tier and session state from that root, so a session that
  changes directory into a worktree it just made keeps its copy key; a
  session with no note uses its payload `cwd`'s worktree.
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
  last-fetched default branch, which nothing refreshes overnight. `DIRTY` (a
  conflicting pull request) is owed as behind too, and `UNKNOWN` reads as not
  read. GitHub reports `BEHIND` only where the base requires branches to be up
  to date, so the live capture's lab base requires it. Only a
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
- **Queued pushes, and the one hand-back** (dcn-2610181035355741, refining
  dcn-2610181016510153). The mode grants no push, so an item whose next act
  is a push (an unpushed branch, a reservation whose work no pull request
  carries, a behind or failing pull request once its local fix is committed)
  may be one the session has no standing instruction to send.
  - **By default a queued push hands nothing back.** The item stays owed and
    the stop stays blocked; a session with no standing instruction to push
    ends at the host's cap, and the stop is recorded for the person.
  - **Only with `--hold-pushes`** does a queued step hand anything back. A
    step naming an own branch, on the same word-boundary match as the link,
    removes only that branch's unpushed item, and only while the disk shows
    the rest clear: its head contains its base's last-fetched tip (the
    default branch, or a stacked branch's base branch as it stands on disk),
    and a preflight receipt names the head (`landing.Receipt`) where the
    repository mints receipts. Where it mints none, the clearing rule's
    equivalent, passing checks in the listing, can only name a pushed
    commit, so nothing is handed back there.
  - **Never handed back:** a pull request that is behind (or `DIRTY`) or
    failing, which the forge reads so until the push; a stacked branch the
    clearing rule still owes; a review outstanding; a reservation named by
    its record id alone; a renewal. So a behind pull request whose merge is
    committed locally stays owed with pushes held and ends at the cap,
    recorded: the instance of 2026-10-16 shows up in the morning instead of
    passing as nothing owed.
  - The conditions are re-read at every check. The stop check records each
    item it hands back (branch and head) in the away record under its lock,
    once per branch and head, for the person's line and `abcd mode back`.
- **What is read, and when (invariant 7,
  adr-2610181003541997).** While the person is present the check reads disk
  only: renewals due (reservations as last fetched, `claim.LastFetched`) and
  unpushed own branches. It never calls the forge, so no turn's end touches
  the network. While away it adds at most one `gh pr list` of the open pull
  requests, every open one when the person's forge login is known (so a
  branch whose pull request another person authored is left out of "own"),
  the person's own alone when it is not (head and base branch, author,
  body, merge state, review decision, check rollup; each field among those
  `internal/ghcontract` pins) under a ten-second deadline. On a failure or
  timeout it fails open to the disk kinds, naming the pull requests as not
  read: a branch with a pull request is compared on disk with the
  last-fetched default branch, naming the fetch time, and a reservation is
  never reported as uncarried, so no duplicate pull request is opened. There
  is no cache: a run of eight blocks costs up to eight
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
message's hash, the count of consecutive blocks, and the wake count with its
own fingerprint of the capped items (below).

**The wake count** (dcn-2610181035361997, refining dcn-2610180936577720 and
dcn-2610181003455731). The capped items are the owed kinds the session can
act on itself: a pull request that is behind or failing, an unpushed branch,
a stacked branch the clearing rule still owes, and a reservation whose work
no pull request carries (which the review's list did not name, but opening
that pull request is the same kind of act as a push). A renewal that is only
due, and an item waiting only on the forge (a stacked branch whose one unmet
condition is its checks still running), are not capped. An allow by row 3
while any capped item is owed adds one to the wake count; a change in the
capped items' fingerprint sets it to zero. The count compares the capped
items alone, so a renewal falling due or done neither resets it nor adds to
it.

While away, the rows are read in order and the first that matches decides:

| # | condition | verdict |
|---|---|---|
| 1 | `last_assistant_message` ends on a line ending in `?`, not yet blocked for this message | block with the reroute's reason (working) |
| 2 | owed, set changed or first block this session | block: next item and count (working) |
| 3 | owed, set unchanged, qualifying wake, and no capped item owed or the wake count below two | allow (waiting; next look kept; the wake count adds one if a capped item is owed) |
| 4 | owed, set unchanged, qualifying wake, a capped item owed and the wake count at two | block: next item; the set has not moved over two wakes, so a wake no longer releases this stop (working) |
| 5 | owed, set unchanged, no qualifying wake | block: next item, and if it cannot move, take another or schedule a look before the deadline (working) |
| 6 | none owed, qualifying wake | allow (waiting; next look kept) |
| 7 | none owed, no qualifying wake | block: take the next ready item first (reserve it), and only if none can be taken alone, schedule a look before the deadline (working) |

So a `*/10 * * * *` cron set once and an unchanged behind pull request give
two waiting spells of up to ten minutes each, then blocks until the item
moves or the host's cap records the stop; each later wake starts a turn that
is blocked the same way, and the identical stops collapse into one recorded
entry with a count.

- **An item that needs a push** (dcn-2610181016510153, refined by
  dcn-2610181035355741). Without `--hold-pushes` the reason says: do the
  local work, then push if your standing instruction allows; queue only if
  the person has said pushes wait, and a push you may not send stays owed,
  so queue it with `abcd mode queue "<step>"` for the person and take
  another item meanwhile. With `--hold-pushes` it says: the person has said
  pushes wait; finish the local work (the merge from the base, the
  preflight), then queue the push naming the branch. It never says the away
  mode allows the push.
- **`stop_hook_active`** decides no row, so both values are covered by every
  row. It alone drives the count (dcn-2610181003484631, refining
  dcn-2610180936577720): a block with it `false` sets the count to one, and
  with it `true` adds one. A change in the owed set no longer resets the
  count, since the host's count does not: in the round-2 scenario a renewal
  falling due at the sixth block would have reset abcd to one, and the host's
  override at its ninth would have found abcd at three, unrecorded. The block
  whose count passes the cap (eight, or `CLAUDE_CODE_STOP_HOOK_BLOCK_CAP`
  when set) records "stopped with work owed" (rows 2, 4 and 5) or "stopped
  with no next look" (row 7) in the away record, labelled "possibly forced by the
  cap", because the host's count also resets on a tool call abcd cannot see.
  The count so errs toward recording a stop that did not happen, never
  toward missing one, and step 7 checks where the host's count resets.
- **Present.** With owed work (the disk kinds only), allow with a
  `systemMessage` naming the work (stopped, flagged); with none, allow
  silently. No forge call.
- **The front door.** `abcd hook stop` maps the payload and writes the host's
  JSON. A malformed payload, an unreadable record or a lock not had allows
  the stop and says so on stderr, following the guard's fail-open-loud
  contract. A stop is never blocked on abcd's own fault. The Stop entry in
  `hooks/hooks.json` carries a timeout of 30 seconds, room for the
  ten-second listing and the git reads.
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

## Settled here

Choices made while building step 1 on 2026-10-18 where this record is
silent, each the narrowest reading that meets the step. Each is the build
lane's choice, recorded for review; none changes a test the step names.

- 2026-10-18 (lane's choice, for review): **Setting the mode again keeps
  what is unread.** `abcd mode autonomous` on a record that holds, or that
  lapsed and is not yet closed, writes a new record (who set it, from which
  copy, a new start and end, `--hold-pushes` as given this time) and carries
  over its queued steps, handed-back items and recorded stops, since nothing
  has printed them yet. The seven-day cap counts from the new start: setting
  the mode is a deliberate act, and the cap is there for a forgotten record.
  A closed record's items, already printed by `abcd mode back`, are not
  carried.
- 2026-10-18 (lane's choice, for review): **`abcd mode queue` needs a record
  that holds.** With none, or one lapsed, it is refused (exit 2) and the
  reason says to ask the person instead, since there is no return to queue
  for. A step's whitespace is folded so it prints as one line, and a step
  over 500 characters is refused, so one step cannot fill the 64 KiB record.
- 2026-10-18 (lane's choice, for review): **Who set it.** The holder name
  reservations resolve with no network read (`claim.ReaderHolder`), else
  the name git would author a commit with (`identity.EffectiveIdentity`,
  which reads `user.name` after git's own overrides), else nothing, shown as
  "someone git does not name". The working copy is its root as git names it,
  shown in the tilde form.
- 2026-10-18 (lane's choice, for review): **Every refusal is exit 2 with
  nothing written**, a lock not had within two seconds included: each is
  "this invocation cannot be honoured", as the badge's refusals are.
- 2026-10-18 (lane's choice, for review): **The lock is checked on every
  read, not only on writes.** A lock file that is a symlink, another
  account's, writable by others or over 64 KiB reads as not away for the
  hooks as well as refusing the verbs, as TestRecordOwnershipFailsOpen
  names. Reading never creates the store; the first write creates its
  folders owner-only.
- 2026-10-18 (lane's choice, for review): **"Shown in the away block" is
  `Record.Terms`.** Step 1 has no away block yet (step 2 prints it), so the
  record's plain statement (until when, set by whom from which copy, that
  pushes wait when `--hold-pushes` is set, that the mode grants no rights,
  and how it ends) is `Record.Terms`, which `abcd mode autonomous` prints and
  step 2's block can repeat.
- 2026-10-18 (lane's choice, for review): **Sibling files go through the
  store.** The created-branch list (step 3) and the session's start-copy
  note (step 2) live in the same folder; `Store.ReadFile` and
  `Store.WriteFile`, called inside `Store.WithLock`, give them the record's
  ownership, size and atomic-write terms. Their names and shapes are those
  steps' to choose.
- 2026-10-18 (lane's choice, for review): **The badge's refusal reads the
  record at the verb.** `mode.RefuseParkWhileAway` is pure and the CLI reads
  the away record for it, so `internal/core/mode` keeps no knowledge of the
  away store; the guard's chip-set path is left to step 6, whose gate denies
  every question while away before that path is reached.
- 2026-10-18 (lane's choice, for review): **`abcd mode` gets its own brief
  chapter**, `04-surfaces/26-mode.md`, beyond the step's named packages.
  Once the verb registers sub-verbs, record-lint's `surface_coverage` rule
  requires a chapter named for it with a sub-verbs table, so the away mode's
  surface text lives there, the register's row 26 points at it, and
  `08-abcd.md` keeps the badge and points at the new chapter. The three
  sub-verbs' sentences and the queue's worked example go in
  `internal/core/surface`, as every visible verb's do.
- 2026-10-18 (step 6 lane's choice, for review): **The away check comes
  before decoding.** "Every question-tool call" is read literally: a
  questions field the check cannot read is refused with the same reason
  while away, rather than running on the loud fail-open status, since a
  question nobody is there to answer stalls the session whatever its shape.
  A payload whose outer JSON does not decode still fails open, because the
  hook then has no working directory to find the record by.
- 2026-10-18 (step 6 lane's choice, for review): **The reason names none of
  the asking rules.** Its head line says the person is away and that no
  question is put to them whatever its chip and whoever wrote it, then
  `autonomy.QuestionRoute`: the queue route with the person-only acts, the
  decide route, and that nothing detects the act. A reason that also listed
  the question's field findings would invite a redraft instead of a
  reroute.
- 2026-10-18 (step 6 lane's choice, for review): **The text lives in
  `internal/core/autonomy`.** `PersonOnlyActs` and `QuestionRoute` are
  additive exports beside the record, so step 2's away block and step 5's
  row 1 ("block with the reroute's reason") can repeat the same list rather
  than a copy that drifts.
- 2026-10-18 (step 6 lane's choice, for review): **An unreadable away record
  is loud at the question gate.** A record or lock the store refuses reads as
  not away, the question is checked as if the person were present, and one
  line names the record: folded into the reason when the gate denies, and
  with exit 1 (the guard's fail-open-loud status) when it would otherwise
  admit silently. Where the gate already answered on stdout with the rows
  note, it keeps exit 0, since exit 1 would drop the note. Outside a
  checkout, with no root commit or no home, nobody is away and nothing is
  said.
- 2026-10-18 (step 6 lane's choice, for review): **The badge's chip-set path
  needs no code of its own.** The away check returns the deny before the
  chip is read, so no mode is set and no question marked open while away;
  TestAwayQuestionSetsNoMode holds it. Beyond the step's named packages,
  `commands/guard.md`, `commands/mode.md` and the guard hook's help text
  gain a sentence each, since each describes the gate and would otherwise
  misstate it while away.
- 2026-10-18 (lane 2's choice, steps 3 and 4, for review): **The
  created-branch list holds a hundred names**, the oldest dropped
  (`autonomy.MaxCreatedBranches`): a hundred names at git's 255-byte limit
  stay well within the store's 64 KiB per file. A rename keeps the record's
  place and time.
- 2026-10-18 (lane 2's choice, for review): **The recorder reads literal
  words only, and runs before the registry loads.** A word a substitution or
  a variable prints makes the command no act; `git -C <dir>` is stepped over,
  not followed, and every name is judged against the session copy's
  repository. It runs ahead of the registry, so a disabled or unloaded
  registry does not stop a branch being recorded. `git worktree add <path>
  <commit-ish>` records the name when it is a branch or resolves to nothing
  (git then creates it), never a tag or an object name; `--detach` and
  `--orphan` record nothing; `git branch -c|-C` is recorded as a creation.
- 2026-10-18 (lane 2's choice, for review): **The copy is the payload
  `cwd`'s worktree until step 2's start-copy note lands.** The recorder
  resolves it in one function (`awayCopyRoot` in `internal/surface/cli`) for
  step 2 to route through its note, and `autonomy.ReadOwed` takes the copy
  root from its caller (`ReadOptions.Root`).
- 2026-10-18 (lane 2's choice, for review): **The away listing lists every
  open pull request, not only the person's,** so a branch whose pull request
  another person authored can be told apart: the person is the holder name
  reservations resolve without a network read (`claim.ReaderHolder`), and
  when that is unknown the listing asks `--author @me` and that rule is not
  applied. At most 200 are listed. The ADR's decision says "the person's
  open pull requests"; invariant 7's clause says "the open pull requests".
- 2026-10-18 (lane 2's choice, for review): **"Recent" is a commit beyond
  the default branch whose author email is this checkout's, made or
  rewritten (the later of its author and committer dates) within 14 days.**
  A live reservation links a branch by its name too (the record id on word
  boundaries), not only through a pull request. The reflog route reads both
  ends of each `checkout: moving from A to B` line and the branch checked out
  now.
- 2026-10-18 (lane 2's choice, for review): **A stacked branch whose base
  has moved on is still found without a pull request:** the branch they share
  a commit with, outside the default branch, is its base when that commit
  was the other branch's own tip by its reflog (set there other than by the
  branch's creation, which tells a base from a branch cut from it). A stacked
  branch is one `stacked` item naming what it needs, never a separate behind
  or failing item; a stacked pull request with no checks reported, where no
  receipts are minted, is not verified and owed.
- 2026-10-18 (lane 2's choice, for review): **When the listing fails, "a
  branch with a pull request" is read as an own branch with an upstream that
  is not gone**, compared on disk with the last-fetched default branch; the
  fetch time is the default branch's reflog, else `claim.LastFetched`.
- 2026-10-18 (lane 2's choice, for review): **A release reservation
  (`rel-…`) is never owed as uncarried work,** since no pull request carries
  a release; its renewal due still counts.
- 2026-10-18 (lane 2's choice, for review): **A queued step names a branch
  as a whole name in which `.`, `_`, `/` and `-` continue the name,** so
  `feat/p` is never found in `feat/p-2`: stricter than the record id's word
  boundary, where `-` ends the id so `fix/iss-5-x` names iss-5. `ReadOwed`
  records each hand-back itself (`Store.RecordHandBack`), and the unpushed
  item's reason is the fact alone; the push wording is the verdict's (step 5).
- 2026-10-18 (lane 2's choice, for review): **The run states are read
  leniently, for their lanes alone**, rather than through the loop's strict
  decoder: a run counts while a spec step is pending or a lane is not done,
  and a lane counts while its stage is not done, discarded or handed back.
  `landing.GHContext` was added beside `landing.GH` to bind the forge client
  to the ten-second deadline, and the guard's `abcd guard` and `abcd guard
  hook` sentences now name the branch record they write.

Choices made while building steps 2 and 5 on 2026-10-18, by build lane 1,
each recorded for review; none changes a test the steps name.

- 2026-10-18 (lane 1's choice, for review): **The start-copy note.** The
  session-start hook notes a session's start worktree in the away store's
  `sessions.json` only in a checkout with `.abcd/` at its root, and only the
  first time for a session id, so a compaction or resume run after the
  session changed into a worktree it made keeps the start. Notes older than
  30 days, and past 256, are dropped. `autonomy.ResolveCopy` reads it back
  for the stop check and for step 3's recorder.
- 2026-10-18 (lane 1's choice, for review): **The prompt hook's read.** It
  reads the away record through the live drain's root-commit lookup, so the
  router still makes one such lookup a prompt, and appends the away block
  after the rules, the memory notes and the status block. The block's words
  come from the owner-only home store, never a tracked file, which is why
  the session-start hook may print it on stdout where its notices stay on
  stderr.
- 2026-10-18 (lane 1's choice, for review): **The person's lines.** While
  the record holds: "abcd: away until <time>, set by <who> from <copy>; N
  steps queued, M items handed back. `abcd mode back` ends it." (with
  "pushes wait for you" under `--hold-pushes`). Lapsed and unclosed: one
  line counting the queued steps and recorded stops, as the plan names, not
  the hand-backs. Both are `Record.PersonLine` and `Record.LapsedLine`.
- 2026-10-18 (lane 1's choice, for review): **The owed seam.** The plan
  names no reader, so the stop hook takes the owed set through one function
  value, `stopOwedReader` in `internal/surface/cli/hook_stop.go`, shaped as
  step 4's `autonomy.ReadOwed` (asked with the start copy, the record, the
  home and the clock; the record not holding means disk only). Until step
  4's reader fills it, it reads nothing owed. Recording a hand-back is
  idempotent per branch and head, so the reader and the front door may both
  record one.
- 2026-10-18 (lane 1's choice, for review): **Two fields on `Owed`.**
  `NextRenewal`, the earliest renewal due among the live reservations under
  this copy's key, due or not yet, which the deadline needs and no owed item
  carries; and `HandedBack`, the items a queued step handed back, which the
  verdict judges. Both are additive.
- 2026-10-18 (lane 1's choice, for review): **The hand-back at the
  verdict.** `Judge` accepts a hand-back only while away, under
  `--hold-pushes`, and only of an unpushed branch's item naming its branch
  and head; any other item the reader hands back is put back among the owed
  items. The front door records each accepted one under the lock.
- 2026-10-18 (lane 1's choice, for review): **Rows the plan leaves open.**
  The prose-question block (row 1) does not take the owed set as seen, so
  the next check names it; at the cap it records nothing, since the plan
  names rows 2, 4, 5 and 7. A block past the cap leaves the session's state
  "stopped". The push sentence goes with an unpushed branch, an uncarried
  reservation, and a behind or failing pull request, the items whose next
  act the plan calls a push. The next look is the earliest-matching
  qualifying wake; a recurring task with no following match by the deadline
  plus an hour is taken as up to 30 minutes late.
- 2026-10-18 (lane 1's choice, for review): **The parser's edges.** Names
  (`MON`), `@` forms and a step on a single value are not read, so they
  never qualify. The day-of-month and day-of-week "written from `*`" rule
  takes a field starting with `*`.
- 2026-10-18 (lane 1's choice, for review): **abcd's own faults allow.**
  Present, the session's state is neither read nor written. Away, a session
  state that cannot be read or saved, and an owed set the reader could not
  return, allow the stop with a stderr line, as a malformed payload does. A
  StopFailure error type outside `[a-z][a-z0-9_]{0,63}` is recorded as
  `unknown`. `hooks/stop.sh` and `hooks/stop-failure.sh` never download the
  binary, as `hooks/subagent-stop.sh` does not, and exit 1 when it is
  missing, never the Stop event's blocking 2; the brief's hook paragraphs
  name them among the exceptions.
- 2026-10-18 (lane 1's choice, for review): **One text for the person-only
  acts.** The away block and the prose-question block carry
  `autonomy.QuestionRoute()`, step 6's statement of the list. Until that
  lane's branch merges, `internal/core/autonomy/question_placeholder.go`
  stands in for its `question.go` with exactly its text; it is deleted at
  the merge.

Choices made while integrating the three lanes on 2026-10-18, each recorded
for review; none changes a test a step names.

- 2026-10-18 (integration's choice, for review): **The placeholder went in
  the merge.** Step 6's `question.go` and lane 1's
  `question_placeholder.go` declared the same names word for word, so the
  placeholder was removed in the merge commit that brought step 6 in, to
  keep every commit building.
- 2026-10-18 (integration's choice, for review): **The stop hook owns the
  hand-back record.** Lane 2's reader recorded a hand-back as it found one,
  and lane 1's front door recorded each the verdict accepted. `ReadOwed` now
  records nothing: it moves the handed-back item from `Owed.Items` into
  `Owed.HandedBack`, the verdict accepts it (`Verdict.HandBacks`), and the
  stop hook records what was accepted under the lock, once per branch and
  head. One owner, and an item the verdict puts back among the owed is never
  recorded as handed back. This replaces lane 2's "`ReadOwed` records each
  hand-back itself" and lane 1's "the reader and the front door may both
  record one" above. `ReadOwed` also fills `Owed.NextRenewal`, present or
  away, from the same halfway point its renewal items use.
- 2026-10-18 (integration's choice, for review): **The seam is wired.** The
  stop hook's `stopOwedReader` defaults to `autonomy.ReadOwed` and is asked
  with `autonomy.ReadOptions`; lane 1's stand-in type is gone.
- 2026-10-18 (integration's choice, for review): **The recorder keys on the
  start copy.** `awayCopyRoot` resolves the copy through
  `autonomy.ResolveCopy` with the hook payload's session id, falling back
  to the payload cwd's worktree when the session has no note or the store
  cannot be opened, so the branch list and the stop check read the same
  copy's key.
- 2026-10-18 (integration's choice, for review): **The ADR says what the
  listing covers.** adr-2610181003541997, still proposed, now says the
  listing covers every open pull request when the person's forge login is
  known, and the person's own alone when it is not, as lane 2's reader does
  and invariant 7's clause now says too; lane 2's note above on the
  mismatch is answered by it.

- 2026-10-18 (fix round 1's choice, for review): **"Managed" reaches a
  linked worktree's main worktree.** Every away hook and the recorder act
  only where `.abcd/` is a directory at the checkout's root or, for a
  linked worktree, at its main worktree's root (`autonomy.Managed`). The
  second half keeps a session that changed into a worktree it made (whose
  branch may predate `.abcd/`, or whose `.abcd/` was never committed)
  checked; one `git rev-parse --git-common-dir` is the only cost, paid only
  in a linked worktree with no `.abcd/`. The recorder now asks the same
  question instead of the local tier, which a fresh worktree lacks.
- 2026-10-18 (fix round 1's choice, for review): **A landed no-upstream
  branch is told by a merge first.** `git cherry` alone misses a squash of
  several commits (its patch matches none of them), so the reader first
  merges the branch into the last-fetched default branch without writing
  anything (`merge-tree --write-tree`, merge drivers replaced as the conflict
  probe replaces them): no change is landed, a change is owed. Only a
  conflicting merge falls to `git cherry`, landed when every commit is
  upstream by patch id, and otherwise a note. So a genuinely unpushed branch
  whose change conflicts with the default branch is a note, not an item: the
  phantom-stop failure the review found is worse than this one, which other
  items (its reservation, uncarried) still cover.
- 2026-10-18 (fix round 1's choice, for review): **The record's parts have
  their own bounds.** Stops past 24 KiB (JSON) let the one seen least
  recently go, and hand-backs past 8 KiB the oldest, each counted
  (`dropped_stops`, `dropped_handed_back`) and printed by `mode back`; the
  queue past 24 KiB refuses the next step (`ErrQueueFull`, exit 2), since a
  queued step is the person's to read and is never cut. Only Begin carrying,
  or Back closing, a record already past 64 KiB lets queued steps go
  (`dropped_queue`), and Back prints every line first. Three fields are
  added to the record, omitted when zero; `schema_version` stays 1.
- 2026-10-18 (fix round 1's choice, for review): **Back replaces what it
  cannot trust.** A record that does not parse, names another schema or is
  not safely this account's (a symlink, writable by others, another
  account's) is replaced by a closed record through the store's atomic
  writer, which replaces a link rather than writing through it, and nothing
  from it is printed. A record with parts of the wrong shape is read for the
  rest. A lock not had within its two seconds, or not safe to take, is gone
  around: the write is atomic, and a re-read catches a writer that held the
  lock and wrote after, closing again (three tries). `autonomous` and `queue`
  still refuse such a record, naming `mode back`.
- 2026-10-18 (fix round 1's choice, for review): **A due renewal's deadline
  is its lapse.** The reader reports `Owed.WakeBy`: each live reservation's
  renewal while not yet due, and its lapse once due, the earliest. The
  deadline reads it less the skew (falling back to `NextRenewal` for a set
  built without it), so a renewal only due releases on a wake that fires
  before the lease lapses, as "Owed work" has it.
- 2026-10-18 (fix round 1's choice, for review): **The budget counts from
  the hook's start.** The owed read has 14 seconds from the hook's start
  (the root lookups included); the forge listing's ten seconds sit inside
  it. A reading cut short sets `Owed.Incomplete`; `Judge` then allows the
  stop with a line naming what was not read and, while away, records it as
  `check-incomplete`. This changes the design in one respect: under heavy
  load a stop with work owed may now be allowed, recorded and shown, where
  before the hook was killed at 30 seconds and the stop went unrecorded.

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
     TestHoldPushesRecorded (`--hold-pushes` is kept in the record, shown in
     the away block, and grants nothing); TestRepeatedStopCollapses (a stop
     identical to the last recorded one adds to its count and last time
     instead of a new entry);
     TestConcurrentWritesUnderLock (two `mode queue` calls racing keep both
     steps, and a stop recorded after `mode back` re-reads inside the lock
     and does not turn the mode back on); TestRecordOwnershipFailsOpen (a
     record or lock that is a symlink, not this account's, writable by
     others, or over 64 KiB reads as not away, and a verb refuses naming
     it); TestBadgeRefusedWhileAway; the badge's three-word vocabulary test
     passes unchanged. The plugin page and the brief's `abcd mode` surface
     text describe the three verbs and `--hold-pushes` and say the mode
     grants no rights, the person's standing instruction deciding what is
     pushed, and `go generate ./internal/surface/cli` refreshes the derived
     appendix. This step also fixes the shapes of `Record` and of `Owed`
     (the owed items, each with its kind, whether it is capped, and its
     branch or record id) in `internal/core/autonomy`, so steps 3 and 4 and
     steps 5 and 6 can run in two lanes against them.
   - landed: 2e859ad4b
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
     every prompt while it holds, naming who set it, from which copy, the
     count of queued steps and of handed-back items, and the literal `abcd
     mode back`); TestSessionStartNotesCopy (the session's start worktree is
     noted per session id, and a later hook run from a worktree the session
     made reads the start copy's key); TestPromptRouterLapsedWindowLine (a
     lapsed, unclosed record with a queued step or a recorded stop shows one
     line naming `abcd mode back` on every prompt until `mode back` closes
     it, and one with neither shows nothing); all inside the hook's existing
     deadline.
   - landed: 472488938
3. The created-branch recorder
   - packages: internal/core/guard, internal/core/autonomy, internal/surface/cli, commands/guard.md, .abcd/development/brief/04-surfaces/17-guard.md
   - tests: internal/core/guard, internal/core/autonomy, internal/surface/cli
   - Watched failing first: TestBranchRecorderOnCreate (`git worktree add -b
     feat/x ../x`, `git switch -c feat/y` and `git checkout -B feat/z`
     record the branch under this copy's key, whether or not the person is
     away, only when no branch of that name exists, and `git checkout -b
     main` on an existing `main` records nothing); TestBranchRecorderOtherForms
     (`git branch feat/w`, a bare `git worktree add ../v` recording `v`, and
     `git worktree add ../u feat/u` on an existing branch are recorded; `git
     branch -m feat/w feat/w2` carries the record; `feat/x` is not matched
     inside `feat/x2`; the file stays within its cap); TestBranchRecorderNeverRefuses
     (the command is allowed whatever the recorder does, and a store it cannot
     write leaves a stderr line); TestBranchRecorderUnderLock (two recordings
     racing keep both). commands/guard.md and the brief's guard chapter say
     the hook records a created branch and refuses nothing for it.
   - landed: 710b9626b
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
     one `DIRTY` is owed as behind; one `CLEAN` is not; `UNKNOWN` reads as
     not read); TestStackedBranchFoundWithoutPullRequest;
     TestStackedBranchClearingRule (owed while behind its base, failing,
     with a review outstanding, or unverified; waiting on its base
     otherwise); TestClearingRuleReview (`CHANGES_REQUESTED` in the listing,
     and a run lane with no passing review of its head, are owed);
     TestClearingRuleWithoutReceipts (in a repository with no receipt
     folder, passing checks verify; with no pull request the reason says it
     was not verified); TestUnpushedSkipsGoneUpstream (a squash-merged
     branch whose upstream is gone is not owed); TestQueuedPushHandsItemBack
     (without `--hold-pushes` a queued step hands nothing back, the
     unpushed branch staying owed; with it, a step naming an unpushed branch
     whose head contains the base's last-fetched tip and has a receipt hands
     back that branch's unpushed item, and the hand-back is recorded once
     per branch and head; with it, none is handed back when the head lacks
     the base tip, has no receipt, or the repository mints no receipts; nor
     a pull request behind, `DIRTY` or failing whose fix is committed
     locally; nor a stacked branch the clearing rule owes; nor one with
     `CHANGES_REQUESTED`; nor a reservation queued by record id alone; nor a
     renewal due; iss-5 queued does not hand back iss-50);
     TestPresentReadsDiskOnly (zero forge calls while present);
     TestForgeListingDeadline (a hung listing ends at ten seconds, and the
     check fails open to the disk kinds with a "not read" note: a branch with
     a pull request is compared on disk naming the fetch time, and a
     reservation is never reported as uncarried);
     TestOwedOrderFixed. Brief invariant 7 gains the away mode's one listing
     per check and its deadline, citing adr-2610181003541997.
   - landed: 1ad31a522
5. The stop verdict, the Stop and StopFailure hooks
   - packages: internal/core/autonomy, internal/surface/cli, hooks
   - tests: internal/core/autonomy, internal/surface/cli
   - Watched failing first: TestJudgeRows (each of the seven away rows and
     both present rows, each with `stop_hook_active` true and false);
     TestWakeReleasesUnchangedSetTwice (with `*/10 * * * *` and an
     unchanged behind pull request, the first two stops with the wake are
     allowed and the third is blocked, and so is each later one; the same
     for failing, unpushed, stacked-not-clear and uncarried items; a set of
     only a renewal due, or of a stacked branch waiting only on running
     checks, keeps releasing; a renewal falling due between the wakes
     neither resets nor adds to the count; a change in the capped items
     resets it; the count and its fingerprint are read back from the
     session's state file by a new hook process);
     TestJudgeEscapeBoundReproducesNoStall (owed, unchanged, `true`, no wake:
     blocked, not allowed); TestJudgeCapRecordsStop (the block passing the
     cap, and an override from the environment, records the stop labelled
     "possibly forced by the cap"); TestCountResetsOnlyOnInactive (a change
     in the owed set at the sixth block does not reset the count, and the
     ninth consecutive block still records the stop; a block with
     `stop_hook_active` false sets it to one); TestPushItemReasonDefersToStandingInstruction
     (without `--hold-pushes` the reason for an unpushed branch says to push
     if the standing instruction allows, to queue only if the person has said
     pushes wait, and that a queued push stays owed; with it, to finish the
     local work and then queue; neither names the away mode as permission); TestDeadlineCeiling (with
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
     TestHookStopOutput (a block only on a block, written since Diff review
     round 1 as the Stop feedback `hookSpecificOutput.additionalContext`, a
     `systemMessage` while present; a malformed payload, an unreadable record
     or a held lock allows with a stderr line); TestHookStopFailureRecords
     (the error type recorded while away, nothing written while present);
     `hooks/hooks.json` gains Stop and StopFailure entries, the Stop entry
     with a 30-second timeout, and `hooks/stop.sh`
     and `hooks/stop-failure.sh` take the same bootstrap and home checks as
     `hooks/subagent-stop.sh` (the hook-shape tests cover them); `go generate
     ./internal/surface/cli` refreshes the derived appendix.
   - landed: 5d754f95b
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
   - landed: 2d27610df
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
     standing instruction, and, with none given and no `--hold-pushes`, the
     push queued and the item still named; with `--hold-pushes` and a
     receipt for its head, the item handed back and the person's line
     counting it; a cron set for the next morning refused as a wake
     and one inside the deadline accepted, with abcd's computed fire time
     compared with the `next run` the host's `CronList` row shows; a
     `*/10 * * * *` cron with the owed set unchanged releasing twice and the
     third stop blocked; nine
     blocks in a row with no tool call, then the same with one tool call in
     between, to see where the host's count resets; a question refused with
     the one reason; the window left to lapse with a queued step, and the
     next prompt's line naming `abcd mode back`; and `abcd mode back`
     printing the queue and the recorded stops. Repeat with the mode off to
     show the stop allowed and flagged with no forge call. Capture before and
     after the change on the same checkout, as the release lesson on host
     fixes requires.
   - landed: d0249f356

## Live captures

Run on 2026-10-18, 13:44 to 14:02 BST, with Claude Code 2.1.302
(`claude-opus-5-5`), against the branch at f403910ac. API spend for all the
runs together: about 1.70 USD.

**The harness.** Everything sits in one scratch folder, `<scratch>`, outside
every working tree:

- `<scratch>/plugin` is `git archive f403910ac`. Its binary rung `abcd` is a
  wrapper that runs `<scratch>/abcd-stall`, which is `go build ./cmd/abcd` at
  f403910ac, with `HOME=<scratch>/home`, so the away store and every other
  file of abcd's lands there. The before capture uses `<scratch>/plugin-before`
  and `<scratch>/abcd-before`, the same build at the branch's merge base
  646f2fb04.
- `<scratch>/repo` is the lab checkout. It holds `CLAUDE.md` with abcd's
  marker block, `.abcd/config.json` and a gitignored local tier. Its bare
  origin is `<scratch>/origin.git`. `feat/x` was made in this checkout with
  `git checkout -b`, so its HEAD reflog holds it, then pushed once, then given
  a second commit by `user.email` that was not pushed. The checkout then went
  back to `main`.
- `<scratch>/pathbin` comes first on PATH. It holds a stub `gh` that logs
  every call and answers `pr list` with `[]`, so no real forge is touched. It
  also holds the same `abcd` wrapper, for the session's own `abcd mode queue`.
- `claude -p` runs with `--plugin-dir`, `--setting-sources project,local`,
  `--output-format stream-json --verbose --include-hook-events`,
  `--no-session-persistence`, `--permission-prompts none`, a narrow
  `--allowedTools` (never `git push`, so no capture shows a push clearing an
  item) and `--max-budget-usd 3` (`<scratch>/run-session.sh`). The runs that
  need several prompts in one session, or `/compact`, use stream-json input
  through `<scratch>/drive.py`, which sends each prompt only after the turn
  before it has ended.
- Transcripts are `<scratch>/transcripts/<n>.jsonl`, with `.stderr` beside
  each, and `<scratch>/summarise.py` turns each into `<n>.summary.txt`.
  Nothing under the account's own Claude configuration was edited.

The person's `!` command and a relayed `!` are not checked here. Those checks
moved to the checklist draft with the guard entries ("Trim after round 2").

**0. Before the change** (`0-before`). The merge-base plugin, the same lab and
the same owed `feat/x`, with the prompt "Say hello and stop." There is no Stop
hook: the session answers "Hello!" and the turn ends (`turns=1`). This is the
stall.

**1. Present, no away record** (`1-present`). The prompt is "Say hello and
stop." SessionStart and the prompt hook add nothing. The Stop hook allows the
stop with `{"systemMessage":"abcd: stopped with work owed (1 item); next:
branch feat/x has commits not pushed …"}`, and the host shows it as "Stop
says: …". The stub `gh` was never called. Verdict: as designed.

**2. Away, no standing instruction to push** (`2-away`). The record was set
with `abcd mode autonomous --until 15:45` (two hours ahead), then the same
prompt was sent.

- SessionStart:startup returns the `## AWAY` block, with "grants no rights",
  the person-only list and "Run `abcd mode back` only on the person's word in
  this session".
- The prompt hook's `systemMessage` is "abcd: away until 2026-10-18 15:45 BST,
  set by Alice from repo; 0 steps queued, 0 items handed back. `abcd mode
  back` ends it.", and the host shows it to the person.
- The first Stop is blocked: `{"decision":"block","reason":"abcd: the person
  is away and work is owed (1 item). Next: branch feat/x has commits not
  pushed … Do the local work, then push if your standing instruction allows;
  queue only if the person has said pushes wait. A push you may not send stays
  owed: queue it with \`abcd mode queue …\`"}`.
- Claude goes on. It says the away mode is no standing instruction and runs
  `abcd mode queue "Push branch feat/x …"`.
- With no `--hold-pushes`, the queued push hands nothing back. The next stop
  is blocked again, adding "If it cannot move, take another or schedule a look
  before 14:15 BST", which is the 30-minute ceiling.
- Every later reply has no tool call, and each one is blocked rather than
  allowed. After ten blocks the host prints "A hook blocked the turn from
  ending 9 consecutive times — overriding and ending turn."
- The stub `gh` logged ten `pr list … --author @me` calls, one per check.
- The away record holds the queued step and one stop, `work-owed`,
  `possibly_forced_by_cap: true`, with count 2. The two are blocks 9 and 10:
  abcd's count ran 1 to 10, since `stop_hook_active` stayed true across the
  tool call.

Verdict: as designed. One block, a tool call, then nine blocks with no tool
call before the host overrode the stop. So **the host's count resets on a
tool call and overrides at the ninth block since**, which is the eighth
continuation. abcd's count does not see that reset, so it records the cap one
block early as well as at the real one. That errs toward recording, as the
spec says.

**3. A question while away.** Print mode does not offer the host's question
tool. `AskUserQuestion` is missing from the session's tool list even with
`--tools AskUserQuestion,Bash,Read`, and missing again in stream-json input
mode. **The live refusal by the host was therefore not captured.**

- *Live* (`3-question`): asked to use AskUserQuestion, the session cannot. It
  follows the away block instead: it decides the choice itself, intends to
  record it with `abcd decide`, queues the push, and is blocked to the cap
  again. That is ten more `gh` calls for ten checks.
- *Hook level* (`3b-question-hook`): an AskUserQuestion PreToolUse payload was
  sent through `sh <scratch>/plugin/hooks/pre-tool-use.sh`, the command the
  host runs. It returns `permissionDecision: "deny"` with "Blocked by the abcd
  guard (question tool): the person is away (abcd mode autonomous), so no
  question is put to them, whatever its chip and whoever wrote it. Route the
  question by the act it asks about …", followed by the full person-only list
  and the route to `abcd decide`.

Verdict: the reroute text is right at the hook. The host side remains to
capture in an interactive session.

**4. Compaction re-injection** (`4-compact`). Stream-json input sent "Say
hello and stop." and then `/compact`.

- The first stop is blocked, and Claude queues the push.
- The second stop is blocked and names the deadline "before 14:19 BST".
  Claude then calls CronCreate with a one-shot `13 14 18 10 *`, and the host
  answers "Next run: Oct 18 2:13pm".
- The third stop is allowed silently. The session state reads `waiting`,
  `next_look` `2026-10-18T14:13:00+01:00` (the host's own next run) and
  `wakes: 1`.
- `/compact` (trigger `manual`) runs PreCompact, and then
  **SessionStart:compact returns the `## AWAY` block again**.

Verdict: as designed, captured live.

**5. Cron wakes, compared with the host** (`5-cron`). Three prompts went to one
session, each sent after the turn before it ended.

- *Turn 1.* CronCreate `0 7 * * *`. The host's CronList row is "Every day at
  7:00 AM (recurring) [session-only], next run Oct 19 7:30am". abcd computes
  the same latest fire time: the 07:00 match plus the host's lateness, at most
  30 minutes. The first stop is blocked (the first block of the session). The
  second, with only the morning cron, is **refused as a wake**: it is blocked
  with "schedule a look before 14:21 BST". Claude then adds `*/10 * * * *`;
  the host lists "next run Oct 18 2:01pm" (13:51 now). abcd's bound is the
  14:00 match plus half the gap, 14:05, before the 14:21 deadline, and the
  host's 14:01 falls inside it. That stop is **allowed** (wake 1).
- *Turn 2*, with the owed set unchanged: allowed (wake 2).
- *Turn 3*: blocked with "The set has not moved over two wakes, so a wake no
  longer releases this stop: move the item, or take another." Then nine
  one-word replies with no tool call. The host overrides at the ninth block,
  and abcd's count reaches nine at that same block and records the stop.
- 13 `gh` calls for 13 checks.

A first attempt sent all three prompts at once. The host merged them into
one turn, which gave nine blocks in a row with no tool call and the host's
override at the ninth. Its transcript was overwritten by the rerun, and its
one recorded stop stays in the record's count.

Verdict: as designed. The `*/10` wake released twice and the third stop was
blocked, and abcd's fire times agree with what the host lists.

**6. `--hold-pushes`, a stacked branch, the hand-back** (`6-hold-stacked`).

- *Setup.* `abcd mode autonomous --until 14:01 --hold-pushes` (eight minutes
  ahead; the queue and the stop were kept). A hand-made receipt file named
  `feat/x`'s head went into `.abcd/.work.local/preflight-receipts/`.
- *Prompt 1.* The session runs `git worktree add -b feat/stacked
  <scratch>/wt-stacked feat/x`. The pre-tool hook records the branch.
- *First stop.* `feat/x` is already handed back: an earlier queued step names
  it, its head contains `origin/main` and has a receipt. Only the new branch
  is owed: "work is owed (1 item). Next: branch feat/stacked has commits not
  pushed … With --hold-pushes the person has said pushes wait: finish the
  local work …, then queue the push naming the branch, as \`abcd mode queue
  "push feat/stacked"\`."
- The stacked condition itself is not owed: the receipt and an equal base tip
  meet the clearing rule. So the item shown is the stacked branch's unpushed
  one.
- Claude queues "push feat/stacked", and the next stop hands that back too.
  Nothing is owed, so row 7 blocks with "take the next ready item first …
  schedule a look before 14:54 BST", which is the one-hour ceiling.
- Claude creates a one-shot cron for 14:37 (host: "Next run: Oct 18 2:37pm"),
  and the stop is allowed.
- *Prompt 2.* The person's line reads "pushes wait for you; 4 steps queued,
  **2 items handed back**". The stop is allowed silently.

Verdict: as designed.

**7. The window lapsed** (`7-lapsed`, after 14:01).

- The prompt hook shows "abcd: the away window ended at 2026-10-18 14:01 BST
  with 4 steps queued and 2 stops recorded, not yet read. `abcd mode back`
  prints them and closes it."
- The stop is checked as present: allowed with "stopped with work owed (2
  items); next: branch feat/stacked has commits not pushed …".
- No `gh` call.

**8. StopFailure** (`8-stopfailure-hook`, hook level). Triggering a real API
error was not possible cheaply, so a payload `{"hook_event_name":
"StopFailure","error":"rate_limit",…}` was sent through `sh
<scratch>/plugin/hooks/stop-failure.sh` while away. It exits 0 with no output,
and the record gains `{"kind":"api-error","error_type":"rate_limit"}`. **A
live StopFailure was not captured**, so the host's field name is not proven
here.

**9. `abcd mode back`** (`9-mode-back`, run by hand). It prints one line per
queued step (four), per handed-back item ("handed back: feat/x at
512f75b1df8a", the same for feat/stacked), and per recorded stop ("stopped
with work owed, possibly forced by the cap: … (6 times, first … 13:46 BST,
last … 13:52 BST)", "stopped on an API error (rate_limit) …"). It ends with
"the away window ended at 2026-10-18 14:01 BST; closed by Alice from repo".

**Not captured, and why.**

- The behind pull request: the stub forge answers `[]`, and a real lab base
  that requires up-to-date branches was out of reach here.
- The host's question refusal live: print mode has no question tool (above).
- A live StopFailure.
- A push clearing an item: pushes were withheld on purpose.

**Seen in passing, none of them blocking.**

- The item's text repeats itself: "branch feat/x has commits not pushed
  (feat/x has commits not pushed)", in the reason, in the person's message
  and in the recorded stop.
- The host marks the first block of each turn with a notification "Stop hook
  error occurred · ctrl+o to see", although the hook exited 0 with outcome
  `success` and empty stderr. That wording is the host's, but a person sees
  it.
- The stops' count overstates the forced stops: 6 recorded against 4 the host
  forced, from the early record when a tool call reset the host's count.
- Bounded by the cap, the blocked session repeats itself. In `3-question` it
  called the hook "faulty" for keeping a queued push owed, which is the
  designed behaviour without `--hold-pushes`.

## Size

Re-estimated after review round 3 at 7.5 working days of work, up from the
5.75 written after the trim (dcn-2610181017056307):

| step | days | after the trim |
|---|---|---|
| 1. The away record, its store, its lock and its verbs | 1.0 | 0.75 |
| 2. Repeated to the agent, shown to the person | 0.75 | 0.5 |
| 3. The created-branch recorder | 0.5 | 0.25 |
| 4. Owed work, and invariant 7's clause | 2.25 | 1.75 |
| 5. The stop verdict, the Stop and StopFailure hooks | 1.75 | 1.5 |
| 6. The question reroute | 0.5 | 0.5 |
| 7. The live session capture | 0.75 | 0.5 |
| total | 7.5 | 5.75 |

Round 3 adds: in step 1 `--hold-pushes`, the collapsed repeated stops and
fixing the `Record` and `Owed` shapes; in step 2 the counts on the person's
line and the session's start-copy note; in step 3 the other branch-creating
forms, the rename and the cap; in step 4 the hand-back's base-tip and
receipt conditions with their real-git cases, `DIRTY` and `UNKNOWN`, and the
fail-open terms; in step 5 the wake count and its row; in step 7 the
hand-back and wake-count checks. The rest of the rise is the Sonnet
reviewer's point that 5.75 days was thin for about 60 tests on real-git
fixtures: that reviewer planned about seven days before these additions, the
Opus reviewer found the elapsed figure credible, and 7.5 is the honest
figure with both.

**Why not four.** About four working days of work is not reachable by
scoping alone. Steps 3 to 5 carry 4.5 days, and every rule in them answers a
review finding: the created-branch, stale-ownership and `Refs:` rules
(round-2 blockers 3 to 5), the merge state (blocker 2), the deadline and
cron cases (blocker 1), and the hand-back and wake count (round-3 F1 and
F2). Cutting any of them reopens a finding, and the stacked-branch rules are
the issue's second instance. The elapsed time can be shorter: `Judge` is
pure over the owed set, so once step 1 fixes `Record` and `Owed`, steps 3
and 4 (2.75 days) run in one lane while steps 2, 5 and 6 (3.0 days) run in
another, which puts the critical path at about 4.75 days with the capture
after both. Whether to run two lanes is the release orchestrator's choice.

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
cap. Review round 3 found that hand-back an exemption the session could
grant itself, and narrowed it to `--hold-pushes` and a verified branch's
unpushed item alone (below).

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

## Review round 3

On 2026-10-18 an Opus and Sonnet pair reviewed the plan as frozen at
600c109d6, with the verdict "ready to build once F1 and F2 are in". They
found the round-2 mapping, the ADR and invariant 7's clause, the recorder
and the two-lane estimate sound. Each item and how it was handled:

| # | finding | handled |
|---|---|---|
| F1 | the queued-push hand-back lets the session exempt itself, replaying the first instance: a behind pull request merged locally and queued was handed back and stayed behind all night; it also covered stacked branches, reservations and renewals, and the person's line did not count it | fixed: by default a queued push hands nothing back, the item stays owed and the cap records the stop; only `abcd mode autonomous --hold-pushes`, which narrows and grants nothing (dcn-2610181016510153 stands), allows a hand-back, and then only of a branch's unpushed item while its head contains the base's last-fetched tip and has a receipt (or the clearing rule's equivalent, which an unpushed head never meets); never a behind, failing or under-review item, a reservation alone or a renewal; the person's line says "N steps queued, M items handed back"; TestQueuedPushHandsItemBack covers each case, the no-flag case included (dcn-2610181035355741) |
| F2 | a cheap wake keeps a stall alive: a `*/10` cron set once releases an unchanged owed set at every wake | fixed: an unchanged set of capped items (behind, failing, unpushed, stacked-not-clear, and an uncarried reservation, added as the same kind of act) is released by a qualifying wake at most twice in a row, then blocked, the cap recording the stop; a renewal only due or an item waiting on running checks keeps releasing; the count and its fingerprint live in the session's state file in the local tier; TestWakeReleasesUnchangedSetTwice; repeated identical stops collapse into one entry so the record stays within 64 KiB (dcn-2610181035361997) |
| minor | the behind kind: `DIRTY` and `UNKNOWN` | handed to the builder, and written into Owed work and TestBehindFromMergeState: `DIRTY` owed as behind, `UNKNOWN` not read |
| minor | the recorder's forms: `git branch <b>`, a bare `git worktree add <path>`, `git worktree add <path> <existing>`, a record carried across `git branch -m` or `-M`, whole-token names, a capped file | handed to the builder, and written into "own" and TestBranchRecorderOtherForms; the cap's bound is the builder's |
| minor | the copy key from a worktree's working directory | handed to the builder, and written into Owed work ("Which copy") and TestSessionStartNotesCopy; the note's shape is the builder's |
| minor | fail-open when the listing fails, stated in the spec and the ADR | handed to the builder, and written into "What is read", TestForgeListingDeadline and adr-2610181003541997: a branch with a pull request compared on disk naming the fetch time, a reservation never reported as uncarried |
| minor | a Stop hook timeout of about 30 seconds | handed to the builder, and written into the front door and step 5 |
| minor | `Record` and `Owed` fixed in step 1 so the lanes can run in parallel | handed to the builder, and written into step 1 and Size |
| size | Sonnet: 5.75 days thin, about seven; Opus: 3.75 days elapsed credible | re-estimated at 7.5 days of work with round 3's additions, about 4.75 elapsed in two lanes (Size) |

No fourth review round: the two deltas are checked on the next commit by
hand.

## Diff review round 1

On 2026-10-18 an Opus and Sonnet pair reviewed the built slice's diff at
f403910ac (against e4cca1e38), with the verdict "fix first", every major
reproduced. The fix round added the release orchestrator's three hard
requirements to majors 1, 3 and 6. Each item and how it was handled:

| # | finding | handled |
|---|---|---|
| major 1 | the Stop hook acted in every git repository, not only managed ones | fixed: Stop, StopFailure, the session-start and prompt away paths and the branch recorder return at once outside a managed checkout (`autonomy.Managed`), with no output, no file under abcd's home, no session note or state, and no git past naming the roots; TestHookStopUnmanagedRepoSilent, TestAwayHooksLeaveUnmanagedRepoAlone (93d34a1ef, e1d64652a) |
| major 2 | a squash-merged branch with no upstream read as unpushed for 14 days | fixed: a merge into the default branch that changes nothing is landed, a conflicting one falls to `git cherry`, and the undecidable is a note; TestNoUpstreamLandedBranchNotOwed, TestNoUpstreamUnlandedBranchOwed, TestNoUpstreamUndecidedBranchNoted (546c22e40) |
| major 3 | a full record kept the person from ending the mode | fixed: AddStop collapses an identical stop anywhere and rotates past its bound; hand-backs rotate; the queue refuses past its bound; Back ends a record past its bound, unparsable, partly unparsable, not safely this account's or behind a busy or unsafe lock, printing what it can and naming the rest; Begin carries a full record fitted; TestAddStopCollapsesAnywhere, TestRecordStaysWithinBoundInUse, TestBackEndsAnyRecord, TestBackWithLockHeld, TestBeginCarriesAFullRecord, TestModeBackEndsAnUnreadableRecord (e101881f2) |
| major 4 | a failed `git worktree add <path> <existing>` recorded a peer's branch | fixed at record time: a name a worktree has checked out is skipped; TestBranchRecorderSkipsBranchCheckedOutElsewhere (fa7f6a8f2). At read time a reflog branch another worktree holds is not own (minor 4's fix); a recorded one stays own, since a worktree the session made holds its branch by design |
| major 5 | a row-1 block past the cap recorded nothing | fixed: work owed when anything is, else no next look; TestRowOnePastCapRecordsStop (e0d16ae76) |
| major 6 | the stop check overran the hook's 30-second timeout | fixed: every git call of the read is bound to a 14-second budget from the hook's start; the stack-base search is linear; a reading cut short allows the stop, named and recorded; TestStackBaseSearchLinear, TestReadOwedHonoursDeadline, TestIncompleteReadAllows, TestHookStopBudgetFailsOpen (31b5ac337). A design change, stated under Settled here |
| 7 | `decision: "block"` shows "Stop hook error occurred" | fixed after reading the hooks reference again (quoted under the host mechanics): a block is `hookSpecificOutput.additionalContext`; TestHookStopBlockIsFeedback (060d25335); the live capture re-run is the orchestrator's |
| 8 | the reason repeated itself | fixed at the source: the reader's reason is the item's words; TestReasonSaysItOnce and the end-to-end exact line (e0d16ae76, e1d64652a) |
| minor 1 | a due renewal never released on a wake | fixed: the deadline reads `Owed.WakeBy`; TestDueRenewalReleasesOnWake builds the set with the reader, and TestWakeReleasesUnchangedSetTwice no longer uses an impossible state (3e4676f32) |
| minor 2 | the 200-listing truncation was silent | fixed: a note, and no reservation reported uncarried from a truncated listing; TestForgeListingAtLimitTruncated (231a6984f) |
| minor 3 | carried counted other people's pull requests | fixed: the carried check applies findOwn's fork and author filter; TestCarriedOnlyByOwnPullRequest (29f62682b) |
| minor 4 | the review-only exclusion leaned on author email | fixed for the case the machine can tell: a reflog branch another worktree has checked out is not own; TestReviewCheckoutSameEmailNotOwn (29f62682b). A sibling's branch no worktree holds any more still reads as own on one email: nothing on disk tells it apart |
| minor 5 | a branch name in a suggested shell command | fixed: each interpolated word is one shell word (`shellquote.Single` unless plain), the renewal's `abcd reserve` too; names were already cleaned by termsafe; TestSuggestedQueueCommandQuotesBranch (e0d16ae76) |
| minor 6 | `hook stop` had no `recover` | fixed: a panic allows the stop with a stderr line, and StopFailure records nothing; TestHookStopPanicAllows (61f1e0e40) |
| minor 7 | overcounted recorded stops | left as the spec allows, labelled "possibly forced by the cap" |
| minor 8 | SubagentStop under `--no-session-persistence` | outside the slice; not handled in this round |
| own | the end-to-end test made its bare origin by hand | fixed: `gittest.InitBare`, which TestNoTestMakesABareRepositoryByHand requires (3b8b14fac) |
