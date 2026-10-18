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
  machine. The session-start hook (every source, `compact` included) and the
  prompt hook repeat it on every prompt while it holds.
- **The stop check.** A plugin Stop hook runs `abcd hook stop`. While the
  record holds, it blocks a turn's end while owed work exists and names the
  next item. It is bounded by the host's `stop_hook_active` flag and by an
  unchanged owed set. While the person is present, it only points the work
  out.
- **The question reroute.** While the record holds, the guard refuses every
  question on the host's question tool. A decision goes to `abcd decide`, and
  a step only the person's hands can do goes to the queue. On their return,
  the queue and the window's decisions are shown once.

Every behaviour is built test-first: each step's tests are written and watched
failing before its code. The last step is a live Claude Code session capture,
because the hook mechanics are the host's and a unit test cannot prove them
(principle evaluate-at-the-user-surface).

## Design

### The host mechanics this rests on

Read on 2026-10-18 at https://code.claude.com/docs/en/hooks (the old
docs.claude.com hooks page redirects there):

- **Stop.** The hook "runs when the main Claude Code agent has finished
  responding" and not on a user interrupt. Its input carries
  `stop_hook_active`, which is "true when Claude Code is already continuing as
  a result of a stop hook". The page says to check it "to avoid blocking on a
  condition that will never resolve". The input also carries
  `last_assistant_message`, `background_tasks` and `session_crons`; the last
  holds the session's scheduled wakes from CronCreate, ScheduleWakeup and
  `/loop`. Output `{"decision": "block", "reason": …}` prevents the stop, and
  the reason "tells Claude why it should continue". Claude Code caps this at
  eight consecutive continuations, and the count resets each time Claude calls
  a tool.
- **SessionStart.** The matcher values include `compact` ("auto or manual
  compaction"). Plain stdout, or `hookSpecificOutput.additionalContext`, is
  added to Claude's context.
- **UserPromptSubmit.** Plain stdout or `additionalContext` is added beside
  the prompt, as a system reminder.
- **Universal output.** `systemMessage` is a "warning message shown to the
  user".

abcd's hook manifest already registers SessionStart with no matcher, so
compaction already runs `abcd hook session-start`. It has no Stop entry.

### The away record

- **Where it lives.** One JSON file per repository on the machine, in abcd's
  machine store under the repository's root commit, beside the worktree store
  (dcn-2610180913399499). It is written through fsutil's atomic writer.
- **What it holds.** Who set it: the holder name reservations cache, without
  a network read, falling back to git's `user.name`. It also holds the start
  and the end (none when open-ended), the queue of steps for the person, and
  the stops recorded with work owed.
- **When it ends.** It holds while now is before its end. It ends with `abcd
  mode back`, or when its time passes. A human prompt never ends it
  (dcn-2610180913399042).
- **The badge.** The three-word badge in `internal/core/mode` is untouched.
  While the record holds, `abcd mode product-thinker` and `abcd mode
  facilitator` are refused, naming `abcd decide`.

### Owed work

Owed work is four kinds (dcn-2610180913416683):

1. a live reservation whose renewal is due or whose work no open pull
   request carries yet;
2. an open pull request of the session's own that is behind the default
   branch or failing its checks;
3. a branch of its own with commits not pushed;
4. a branch of its own stacked on a change that is still open. Its merge from
   the default branch, fixes, review and preflight are owed now, and only its
   landing waits for the base.

- **Which branches are the session's own.** A branch is the session's own when
  the working copy's own HEAD reflog has had it checked out since the window
  began. Sibling working copies share the branch namespace, so this keeps one
  copy from being held for another's branch.
- **What is read from disk.** Reservations are read as last fetched
  (`claim.LastFetched`). "Behind" is measured against the last-fetched default
  branch.
- **What is read from the forge.** One `gh pr list` of the person's open pull
  requests per stop check gives the head and base branch, the merge state and
  the check rollup. Brief invariant 7 gains this as the away mode's documented
  meaning, as the drain's listing is. If the listing fails, the check uses the
  disk kinds only and says the pull requests were not read.
- **Order.** The next item is the first of: a renewal due, a failing pull
  request, a behind one, a stacked branch, an unpushed branch, unhanded work.

### The stop verdict

`autonomy.Judge(record, owed, payload) Verdict` is pure:

| state | owed | `stop_hook_active` | verdict |
|---|---|---|---|
| away | some, changed or first time | any | block: next item and count (working) |
| away | some, unchanged since this session's last block | true | allow; record "stopped with work owed" |
| away | none, a wake or background task present | any | allow (waiting) |
| away | none, no wake | false | block once: schedule the next look within half the lease |
| away | none, no wake | true | allow; record it |
| present | some | any | allow; `systemMessage` names the work (stopped, flagged) |
| present | none | any | allow, silent |

- **The fingerprint.** "Unchanged" compares a hash of the owed set with the
  one kept for the session id in the local tier.
- **The front door.** `abcd hook stop` maps the payload and writes the host's
  JSON. A malformed payload or an unreadable record allows the stop and says
  so on stderr, following the guard's fail-open-loud contract. A stop is never
  blocked on abcd's own fault.

### The question reroute

`questionGate` (`internal/surface/cli/guard_question.go`) reads the away
record first. While it holds, it denies every question-tool call, abcd's or
another tool's. The deny reason depends on the chip:

- a Product or Tech chip, or no chip, gets the reason to decide and record it
  with `abcd decide`, with grounds and how to reverse it;
- a Setup chip gets the reason to queue the step with `abcd mode queue`.

No mode is set and no question is marked open (dcn-2610180913431493).

### Shown

- **On the person's return.** The prompt hook's first run after the record
  ends prints the queue, the window's decisions and the recorded stops once,
  then marks the record shown. The window's decisions are the records whose
  ids fall between its start and end (dcn-2610180913431985).
- **On the status line.** It reads the record and names the away mode and its
  end.
- **From the command line.** A bare `abcd mode` prints the badge, the record,
  the queue and the owed items.

## Footprint

- packages: internal/core/autonomy, internal/core/mode, internal/core/statusline, internal/surface/cli, hooks, commands/mode.md, .abcd/development/brief/02-constraints/03-invariants.md, .abcd/development/brief/04-surfaces/08-abcd.md, .abcd/development/brief/04-surfaces/17-guard.md
- tests: internal/core/autonomy, internal/core/mode, internal/core/statusline, internal/surface/cli

## Steps

1. The away record and its verbs
   - packages: internal/core/autonomy, internal/surface/cli, commands/mode.md
   - tests: internal/core/autonomy, internal/surface/cli
   - Watched failing first: `abcd mode autonomous --until 09:00` writes one
     record that a sibling worktree of the same repository reads; `--until`
     takes a clock time (its next occurrence) or a full timestamp; a past time,
     no local tier or no checkout is refused with nothing written; `abcd mode
     back` ends it; a record past its end reads as ended; `abcd mode queue`
     appends; `abcd mode product-thinker` is refused while the record holds;
     the badge's three-word vocabulary test still passes unchanged.
2. Repeated at start, compaction and every prompt
   - packages: internal/surface/cli
   - tests: internal/surface/cli
   - Watched failing first: `abcd hook session-start` with source `startup`,
     `resume` and `compact` prints the away block while the record holds and
     nothing when it does not; `abcd hook prompt-router` adds it on every
     prompt, including a second prompt in the same session, outside the
     loader's dedup ledger; both stay inside the hook's existing deadline.
3. Owed work
   - packages: internal/core/autonomy
   - tests: internal/core/autonomy
   - Watched failing first, against real git repositories in temp folders
     and a fake forge listing pinned to `internal/ghcontract`'s fields: each
     of the four kinds is found; a sibling worktree's branch is not the
     session's own; a stacked branch with an open base is owed and its item
     says only the landing waits; a reservation whose pull request is open and
     green is not owed until its renewal is due; a failed listing yields the
     disk kinds and a "not read" note; the order of items is fixed.
4. The stop verdict and the Stop hook
   - packages: internal/core/autonomy, internal/surface/cli, hooks
   - tests: internal/core/autonomy, internal/surface/cli
   - Watched failing first: every row of the verdict table; the fingerprint
     is kept per session id; `abcd hook stop` writes `{"decision":"block",
     "reason":…}` only on a block and a `systemMessage` while present; a
     malformed payload allows the stop with a stderr line; `hooks/hooks.json`
     gains a Stop entry and `hooks/stop.sh` takes the same bootstrap and home
     checks as `hooks/subagent-stop.sh` (the hook-shape tests in
     `internal/surface/cli` cover it).
5. The question reroute and the person's return
   - packages: internal/surface/cli, internal/core/autonomy
   - tests: internal/surface/cli, internal/core/autonomy
   - Watched failing first: while away, a Product-chip, a Tech-chip and an
     unchipped question are denied with the decide reason, and a Setup-chip
     question with the queue reason; another tool's question is denied too;
     no mode is set and no marker written; while present the gate behaves
     exactly as today. The first prompt after the record ends prints the
     queue, the window's decisions (by id range) and the recorded stops,
     once.
6. Shown, documented, and invariant 7's clause
   - packages: internal/core/statusline, internal/surface/cli, commands/mode.md, .abcd/development/brief/02-constraints/03-invariants.md, .abcd/development/brief/04-surfaces/08-abcd.md, .abcd/development/brief/04-surfaces/17-guard.md
   - tests: internal/core/statusline, internal/surface/cli
   - Watched failing first: the status line names the away mode and its end;
     a bare `abcd mode` prints the record, the queue and the owed items with
     no host. Brief invariant 7 gains the stop check's one forge listing as
     the away mode's documented meaning. The plugin page and the brief's
     surface chapters say what each verb does. Then `go generate
     ./internal/surface/cli` is run for the derived appendix.
7. The live session capture
   - packages: none (evidence only, kept in the local tier)
   - tests: a live Claude Code session with the plugin built from the branch
   - In a lab checkout, seed one behind pull request, one stacked branch and
     one unpushed branch, set `abcd mode autonomous --until` an hour ahead,
     and record the transcript. It must show the away block on start; a
     manual `/compact` followed by the block again; an attempted stop blocked
     naming the behind pull request; the stacked branch's item; a question
     refused with the decide reason; `abcd mode back` showing the decisions
     and the queue. Repeat with the mode off to show the stop allowed and
     flagged. Capture before and after the change on the same checkout, as
     the release lesson on host fixes requires.
