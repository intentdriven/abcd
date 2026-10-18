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
(itd-2610170638184486) and the fix for iss-2610170619247185. Three parts:

- **The away mode.** `abcd mode autonomous [--until <time>]`, `abcd mode back`
  and `abcd mode queue "<step>"` keep one away record per repository on the
  machine, changed only under a lock. The session-start hook (every source,
  `compact` included) and the prompt hook repeat it to the agent while it
  holds, and the prompt hook shows the person one line saying so.
- **The stop check.** A plugin Stop hook runs `abcd hook stop`. While the
  record holds, a session is only working or waiting: a turn's end is blocked
  while owed work exists, naming the next item, and is allowed only with a
  wake that fires before the session's reservations need renewing. The host's
  cap of eight consecutive continuations is the outer bound. A StopFailure
  hook records a turn the host ends on an API error. While the person is
  present the check reads disk only and points the work out.
- **The question reroute.** While the record holds, the guard refuses every
  question on the host's question tool. An act only the person may approve
  goes to the queue; every other question is decided and recorded with `abcd
  decide`. `abcd mode back` prints the queue and the recorded stops.

Every behaviour is built test-first: each step names its tests, written and
watched failing before its code. The last step is a live Claude Code session
capture, because the hook mechanics are the host's and a unit test cannot
prove them (principle evaluate-at-the-user-surface).

Moved out by review round 1 (dcn-2610180937181929), back to the checklist
draft as named remaining work: the report shown once on the person's return
(the queue, the window's decisions gathered across working copies, the
recorded stops and who ended the mode), the status line, the bare `abcd mode`
print, and the brief's surface text for them.

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

abcd's hook manifest already registers SessionStart with no matcher, so
compaction already runs `abcd hook session-start`. It has no Stop or
StopFailure entry, and its PreToolUse entry already matches the host's
question tool.

### The away record

- **Where it lives.** One JSON file per repository on the machine, in abcd's
  machine store under the repository's root commit, beside the worktree store
  (dcn-2610180913399499), with a lock file beside it. It is written through
  fsutil's atomic writer.
- **What it holds.** Who set it: the holder name reservations cache, without
  a network read, falling back to git's `user.name`. It also holds the start,
  the end, the queue of steps for the person, the stops recorded (with work
  owed, with no next look, or on an API error with its type), and, once ended
  by word, who ended it and from which working copy.
- **The lock** (dcn-2610180937181736). Every read-modify-write (`mode
  autonomous`, `mode back`, `mode queue`, a recorded stop) holds
  `fsutil.WithFileLock` on the lock file and re-reads the record inside it.
  The stop check reads the record and lists the forge outside the lock, then
  takes it, re-reads, and records its stop only if the record still holds, so
  a check that began while away never writes away back after `abcd mode
  back`. A lock not had within two seconds: the verbs refuse naming it, and
  the hooks allow the stop and say so on stderr.
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
  one-line `systemMessage` on every prompt: away until when, set by whom, and
  that saying they are back ends it. The away block tells the agent to run
  `abcd mode back` only on the person's word in its session.
- **Ending it.** `abcd mode back` prints the queue and the recorded stops,
  records who ended the mode and from which working copy, and ends the
  record; on a record whose time has passed it prints the same and closes it.
- **The badge.** The three-word badge in `internal/core/mode` is untouched.
  While the record holds, `abcd mode product-thinker` and `abcd mode
  facilitator` are refused, naming `abcd decide`.

### Owed work

Owed work is four kinds (dcn-2610180913416683, refined by
dcn-2610180936568358 and dcn-2610180936571036):

1. a live reservation under this copy's key whose renewal is due, or whose
   work no open pull request carries yet;
2. an open pull request of the session's own that is behind the default
   branch or failing its checks;
3. a branch of its own with commits not pushed;
4. a branch of its own stacked on a change that is still open, while it
   needs work (the clearing rule below).

- **What "own" means.** A branch other than the default branch is the
  session's own when any of these holds, with no window bound:
  - this working copy's HEAD reflog has checked it out at any time;
  - it is the branch of a lane in a run state this copy's local tier holds
    (`.abcd/.work.local/run/<run-id>/state.json`, whose lanes `build` and
    `drain` cut with `git worktree add`, so they never reach this copy's
    reflog);
  - it is the head branch of an open pull request linked to a reservation
    under this copy's key.

  A sibling copy's branch is none of these, so it never holds this session.
- **The reservation filter and link.** A reservation counts only when its
  copy key is this copy's (the view's `Copy == CopyThis`). A pull request
  carries a reservation's work when the record id appears in its head
  branch's name or on a `Refs:`, `Resolves:` or `Delivers:` line of its body.
  A renewal is due halfway between the reservation's latest claim or renewal
  and its expiry.
- **Finding a stacked branch.** An own branch is stacked when its open pull
  request's base is not the default branch, or, with no pull request, when
  the nearest other own branch whose tip is an ancestor of its tip is not
  contained in the last-fetched default branch.
- **The clearing rule for a stacked branch.** One whose base is still open is
  owed only while it is behind its base's tip or behind the last-fetched
  default branch, its checks are failing, or no preflight receipt names its
  head in any worktree (`landing.Receipt`). Otherwise it is waiting on its
  base, which is not owed: only its landing waits.
- **Unpushed.** An own branch with commits not on its upstream, or, with no
  upstream configured, not in any remote-tracking ref. A branch whose
  upstream is gone (merged and deleted on the remote, a squash merge
  included) never counts.
- **What is read, and when (invariant 7).** While the person is present the
  check reads disk only: renewals due (reservations as last fetched,
  `claim.LastFetched`) and unpushed own branches. It never calls the forge,
  so no turn's end touches the network. While away it adds at most one `gh pr
  list` of the person's open pull requests (head and base branch, body, merge
  state, check rollup) under a ten-second deadline, cached in the local tier:
  a listing younger than ten minutes is reused, not repeated, so eight blocks
  in a row cost one call; on a failure or timeout a cached listing younger
  than half the default lease is used with its age named, else the disk
  kinds alone with the pull requests named as not read. "Behind" is measured
  against the last-fetched default branch. Brief invariant 7 gains this
  listing as the away mode's documented meaning, as the drain's listing is.
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

**The qualifying wake** (dcn-2610180936579498). A `session_crons` entry
qualifies when the next time its schedule matches after now, read in the
machine's local time, is no later than the deadline: the earliest renewal due
among the live reservations under this copy's key, minus `lease.Skew`; or,
with none held, now plus half of `lease.Default`. The next fire time comes
from a five-field cron parser of abcd's own (minute, hour, day of month,
month, day of week; lists, ranges, steps and `*`; when both day fields are
restricted, either matches, as cron does), which steps minute by minute from
now to the deadline, at most a day's minutes since `lease.Max` is a day. An
expression it cannot parse never qualifies. A background task never counts
on its own while away. No new dependency.

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

- **`stop_hook_active`** decides no row, so both values are covered by every
  row, the old missing row (owed, unchanged, `false`) included. It drives the
  count: a block with it `false` sets the count to one, and with it `true`
  adds one; a changed fingerprint also resets it. The block whose count
  passes the cap (eight, or `CLAUDE_CODE_STOP_HOOK_BLOCK_CAP` when set)
  records "stopped with work owed" (rows 2 and 4) or "stopped with no next
  look" (row 6) in the away record, since the host will override it. The
  host's count resets on a tool call abcd cannot see, so abcd's count errs
  toward recording a stop that did not happen, never toward missing one.
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
agent routes it:

- **Queued for the person** with `abcd mode queue "<step>"`, carrying on with
  other work meanwhile, is a question about an act only the person may
  approve:
  - an approval the repository binds to the person per act: an embargo
    close's `--approve <digest>`, which `commands/embargo.md` says is passed
    only after the person, in this session, has read the text and said yes,
    and never supplied on the agent's own; a release sign-off;
  - publishing anything, or changing a repository's visibility;
  - adding a dependency;
  - a secret, a credential or an App;
  - a step only a website or a browser can do;
  - a destructive act: deleting what the session did not create, overwriting
    uncommitted work, a force push, discarding a peer's work;
  - anything that spends money.
- **Decided and recorded** with `abcd decide`, with its grounds and how to
  reverse it, is every other question.

The away mode, set on the person's word, is their ask for the session to
commit and push its own branches and open their pull requests, which is the
owed work; merging follows the repository's own rule. The away block repeats
the list. No mode is set and no question is marked open.

## Footprint

- packages: internal/core/autonomy, internal/core/mode, internal/surface/cli, hooks, commands/mode.md, .abcd/development/brief/02-constraints/03-invariants.md, .abcd/development/brief/04-surfaces/08-abcd.md, .abcd/development/brief/04-surfaces/17-guard.md
- tests: internal/core/autonomy, internal/core/mode, internal/surface/cli

## Steps

1. The away record, its lock and its verbs
   - packages: internal/core/autonomy, internal/core/mode, internal/surface/cli, commands/mode.md, .abcd/development/brief/04-surfaces/08-abcd.md
   - tests: internal/core/autonomy, internal/core/mode, internal/surface/cli
   - Watched failing first: TestAwayRecordSharedAcrossWorktrees (`abcd mode
     autonomous --until 09:00` writes one record a sibling worktree reads);
     TestUntilClockTimeAndTimestamp; TestUntilRefusedPastOrNoTierOrNoCheckout
     (nothing written); TestStaleRecordCapSevenDays (an open-ended record
     reads as ended seven days on, and `--until` beyond that is refused);
     TestModeBackPrintsQueueAndStopsAndRecordsEnder (also on a record whose
     time has passed); TestRecordPastEndReadsEnded; TestQueueAppends;
     TestConcurrentWritesUnderLock (two `mode queue` calls racing keep both
     steps, and a stop recorded after `mode back` re-reads inside the lock and
     does not turn the mode back on); TestRecordOwnershipFailsOpen (a record
     or lock that is a symlink, not this account's, writable by others, or
     over 64 KiB reads as not away, and a verb refuses naming it);
     TestBadgeRefusedWhileAway; the badge's three-word vocabulary test passes
     unchanged. The plugin page and the brief's `abcd mode` surface text
     describe the three verbs, and `go generate ./internal/surface/cli`
     refreshes the derived appendix.
2. Repeated to the agent, shown to the person
   - packages: internal/surface/cli
   - tests: internal/surface/cli
   - Watched failing first: TestSessionStartAwayBlockEverySource (`startup`,
     `resume` and `compact` print the away block while the record holds,
     nothing when it does not, and the block carries the person-only act list
     and the rule to run `abcd mode back` only on the person's word);
     TestPromptRouterAwayBlockEveryPrompt (a second prompt in the same session
     too, outside the loader's dedup ledger); TestPromptRouterAwaySystemMessage
     (one line the person sees on every prompt while it holds); both inside
     the hook's existing deadline.
3. Owed work, and invariant 7's clause
   - packages: internal/core/autonomy, .abcd/development/brief/02-constraints/03-invariants.md
   - tests: internal/core/autonomy
   - Watched failing first, against real git repositories in temp folders
     and a fake forge listing pinned to `internal/ghcontract`'s fields:
     TestOwnBranchFromReflogWithoutWindow (checked out before the mode was
     set, still own); TestOwnBranchFromRunLane (a lane cut with `git worktree
     add`, absent from this copy's reflog, is own);
     TestOwnBranchFromLinkedReservation; TestSiblingBranchNotOwn;
     TestReservationCopyKeyFilter (a sibling copy's live reservation is not
     owed); TestReservationPullRequestLink (record id in the head branch or a
     `Refs:`/`Resolves:`/`Delivers:` body line; a green linked pull request is
     not owed until renewal is due); TestStackedBranchFoundWithoutPullRequest;
     TestStackedBranchClearingRule (owed while behind its base or the default
     branch, failing, or without a receipt; waiting on its base otherwise);
     TestUnpushedSkipsGoneUpstream (a squash-merged branch whose upstream is
     gone is not owed); TestPresentReadsDiskOnly (zero forge calls while
     present); TestForgeListingDeadlineAndCache (a hung listing ends at ten
     seconds; a listing younger than ten minutes is reused; a stale cache
     falls back with its age named, else the disk kinds and a "not read"
     note); TestOwedOrderFixed. Brief invariant 7 gains the away mode's one
     listing per check, its deadline and its cache.
4. The stop verdict, the Stop and StopFailure hooks
   - packages: internal/core/autonomy, internal/surface/cli, hooks
   - tests: internal/core/autonomy, internal/surface/cli
   - Watched failing first: TestJudgeRows (each of the six away rows and both
     present rows, each with `stop_hook_active` true and false);
     TestJudgeEscapeBoundReproducesNoStall (owed, unchanged, `true`, no wake:
     blocked, not allowed); TestJudgeCapRecordsStop (the block passing the
     cap, and an override from the environment, records the stop);
     TestCronNextFire (lists, ranges, steps, both day fields, an unparsable
     expression never qualifies); TestQualifyingWakeDistantCron (`0 9 * * *`
     seen at 21:00 does not qualify); TestQualifyingWakeRecurringCron (`*/30 *
     * * *` does); TestQualifyingWakeAfterLeaseLapse (a wake past the earliest
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
5. The question reroute
   - packages: internal/surface/cli, internal/core/autonomy, .abcd/development/brief/04-surfaces/17-guard.md
   - tests: internal/surface/cli, internal/core/autonomy
   - Watched failing first: TestAwayQuestionDeniedOneReason (a Product-chip,
     a Tech-chip, a Setup-chip and an unchipped question, and another tool's,
     get the same reason); TestAwayReasonListsPersonOnlyActs (the reason
     names every act in the list, the embargo `--approve` digest among them,
     and the route to `abcd mode queue` for those and to `abcd decide` for
     the rest); TestAwayQuestionSetsNoMode (no mode set, no marker written);
     TestPresentQuestionGateUnchanged. The brief's guard chapter says what
     the gate does while away.
6. The live session capture
   - packages: none (evidence only, kept in the local tier)
   - tests: a live Claude Code session with the plugin built from the branch
   - In a lab checkout, seed one behind pull request checked out before the
     mode is set, one stacked branch made with `git worktree add` by a run,
     and one unpushed branch; set `abcd mode autonomous --until` an hour
     ahead, and record the transcript. It must show the away block on start
     and the person's line; a manual `/compact` followed by the block again;
     an attempted stop blocked naming the behind pull request; a reply with
     no tool call blocked again rather than allowed; the stacked branch's
     item; a cron set for the next morning refused as a wake and one inside
     the deadline accepted; a question refused with the one reason; `abcd
     mode back` printing the queue and the recorded stops. Repeat with the
     mode off to show the stop allowed and flagged with no forge call.
     Capture before and after the change on the same checkout, as the
     release lesson on host fixes requires.

## Size

Re-estimated in review round 1 at five working days, the risk sitting in
step 3's reflog, run-state and forge reading and step 4's cron parser:

| step | days |
|---|---|
| 1. The away record, its lock and its verbs | 0.75 |
| 2. Repeated to the agent, shown to the person | 0.5 |
| 3. Owed work, and invariant 7's clause | 1.5 |
| 4. The stop verdict, the Stop and StopFailure hooks | 1.25 |
| 5. The question reroute | 0.5 |
| 6. The live session capture | 0.5 |
| total | 5.0 |

The first plan wrote no size down; the four-day figure the review was asked
to check appears nowhere in it, and it would have covered steps 1 to 5 of
that plan without the review's fixes.

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
