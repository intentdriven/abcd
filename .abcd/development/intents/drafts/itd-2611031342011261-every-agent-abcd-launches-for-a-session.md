---
id: itd-2611031342011261
slug: every-agent-abcd-launches-for-a-session
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609201916056194, itd-166, itd-2610111707085098, itd-2611011230226453]
related_issues: [iss-2611020719494343, iss-2611031146255825, iss-2611031337115764, iss-2610100913096346]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# abcd keeps a per-session ledger of every agent it launches

## Press Release

> **Every agent abcd starts is written down against the session that started it, and the session can add them up.** A person leaves a build running and comes back to a status line that says how many agents are running of how many it started, how many failed, what the key-billed ones have spent against each key's budget, and how many tokens the lot have used; a line a person can ask about, and an agent can read, says where each figure came from. Behind the row is one ledger on the person's own machine: one entry per agent abcd launched, naming the session or terminal run that asked for it, the piece of work it was for, whether it ran on a named key or on the subscription, the model that answered, what it spent, its tokens, the time it spent waiting on the model, how it ended, and whether it is still running. Money is counted only for agents billed to a key; subscription work shows in the subscription's own gauges and is never added to the bill. A run left to work on its own gives the same account of itself when it is done: every agent it started, with its cost and outcome, in one listing. "I used to keep a spreadsheet of what each overnight run had cost," said Maya, an autonomous-development practitioner. "Now I ask abcd, and the answer covers the agents I never saw start."

_The draft's seeded wording (2026-11-03, from the lab) cited the findings by number; it is revised above to the user moment, with the evidence moved to Why This Matters._

## Why This Matters

Typed links: `builds_on`
[itd-2609201916056194](../shipped/itd-2609201916056194-abcd-runs-a-delegated-agent-through-a.md)
(the runner, the one seam every launch abcd makes passes through),
[itd-166](../shipped/itd-166-each-run-should-record-the-model.md) (both
sides of a run: the model asked for and served, the tokens and the cost, kept
on the run state today),
[itd-2610111707085098](../shipped/itd-2610111707085098-the-status-line-should-show-which.md)
(each session's persona name, which the ledger names a session by) and
[itd-2611011230226453](itd-2611011230226453-when-abcd-starts-agents-on-your-own-api.md)
(the key a session names and the budget it keeps; this ledger is where each
launch's key and spend are written, so the two share one record of what was
spent). Related issues: the run-state schema trap
([iss-2611020719494343](../../../work/issues/open/iss-2611020719494343-a-lane-whose-change-raises-the-run-state.md)),
which the ledger's own schema is shaped to avoid; the unattended build drive
([iss-2611031146255825](../../../work/issues/open/iss-2611031146255825-no-abcd-verb-drives-an-intent-build-to.md)),
whose account of itself this ledger is; the stranded agent
([iss-2611031337115764](../../../work/issues/open/iss-2611031337115764-a-finished-agent-is-stranded-when-the.md)),
which the ledger shows as two launches for one await; and the run-start
budget check that always skips
([iss-2610100913096346](../../../work/issues/open/iss-2610100913096346-the-run-start-budget-check-never.md)),
which a per-key sum of recorded launches answers. The status line's own
elements, words and dropping order are the sibling draft's,
[itd-2611031342000215](itd-2611031342000215-the-status-line-names-one-role-at-a-time.md);
this intent supplies the figures that draft's agents and API elements show.

abcd starts agents on a person's behalf and keeps no account of them as a
set. The lab of 2026-11-03 (lab-261103114820-bcba06f) read what the record
holds today and what the host can see:

- The run state keeps each launch's spend and tokens on its route record but
  no API time, so the time the agents spent waiting on the model cannot be
  summed from it (finding F-2); and nothing in it names the session that
  drove the run, so a session cannot find the runs it started, let alone sum
  them (F-3). The planning interviewer's and the drain judge's launches go
  through the same runner and leave no per-launch figures anywhere.
- The host's own counters see none of it: the coordinating session of that
  day started 36 background shells that launched agents, and its transcript
  carries no trace of them, so the host's cost and token figures cover none
  of those agents; only a ledger abcd keeps can count them (F-4). The person
  kept one by hand, a spend log per key in a scratch file, which the
  prototype on their status line read; on the night of 2026-11-01 three
  sessions each tallied their own spend by script and had to ask each other
  what was left of the key.
- A subscription session's cost figure is an estimate of what the work would
  have cost, not money spent: adding it to the key-billed agents' spend
  overstated the bill (F-12). Which route a launch billed is recorded
  nowhere, and the harness's result does not say; the runner that chose the
  route has to write it down.
- Summed over that session, 70 agent results came to 48.2 million tokens,
  almost all cache reads, so a token figure needs a stated basis (F-5).
- On 2026-11-03 a routed implementer ran for an hour and a half, committed
  and wrote its receipt, and the step that would have collected it was
  refused; every later step asked for a fresh implementer, and an unattended
  run would have paid for the work twice (iss-2611031337115764). A ledger
  that shows two launches for one await is the first place that would have
  been visible.

The unattended runs abcd is moving toward (the drain from a plain terminal,
itd-2610222215485265; a build driven to done, iss-2611031146255825; the
autonomy grant's report, itd-2610240850336695) each need to say what they
started and what it cost, and today each would have to reconstruct that from
run states in several worktrees' local tiers, which a worktree's removal
deletes.

## Mechanism

- We expect a session, and a run left on its own, to account for every agent
  abcd started for it, because every launch passes through one dispatch seam
  (the runner's dispatcher, which the build, the drain, its judge and the
  planning interviewer all call) that writes the entry before the harness
  starts and completes it when the harness ends, keyed on the session the
  host exports to the command, so no launch can escape the record and no sum
  depends on a process pattern or on the host's own counters, which see
  none of them. Shown wrong if a launch abcd made is found with no entry, or
  if the number of running agents the ledger reports disagrees with the
  processes abcd holds handles for.

## Scope Conditions

- Agents abcd launches itself through a runner (the claude and opencode
  runners, and a configured fallback host): a build or drain lane's
  implementer and reviewers, the drain's judge, the planning interviewer, a
  review abcd starts. A role handed to the host session is the host's own
  agent, counted by the host, and gets no figures of its own; an agent a
  person or a session starts by hand in a shell is not abcd's launch and
  gets no entry.
- The orchestrating session's own figures (its cost, API time and tokens)
  are the host's payload's and its transcript's, not the ledger's; the ledger
  is the agents' side, and a status line adds the session's own cost to the
  bill only when the session itself bills a key.
- One machine: the ledger lives under abcd's home on the machine that
  launched, and the same session or key on a second machine has a ledger of
  its own there. A hosted runner's home is as ephemeral as the runner, so
  there the run's own summary is the account that survives.
- Cost figures are the harness's own estimate at its price table; the
  provider's invoice is the final figure, and every render that shows spend
  says so. A harness that reports no API time leaves that field absent.
- Which route billed a launch is what abcd handed it or what the harness's
  event stream shows, never an account abcd looked up: a launch abcd handed
  a named key is key-billed; one whose stream reports rate-limit windows is
  the subscription's; one abcd can place in neither is recorded as unknown
  and counted in no sum of money.

## Acceptance Criteria

Each is a proposal for the product thinker to confirm at the planning
interview, drafted from the lab's rulings of 2026-11-03.

- **Given** any agent abcd launches through a runner (a build or drain lane's
  implementer or reviewer, the drain's judge, the planning interviewer, a
  review abcd starts), **when** the harness starts, **then** one entry exists
  in the ledger naming the orchestrating session, the repository, the run,
  lane and role (or the interview and turn), the runner, the model asked for
  and the billing route, marked running; and **when** the harness ends,
  **then** the same entry carries the model that answered, the tokens by
  kind, the cost and its unit, the API time and wall time, the outcome
  (finished, failed with its reason, stopped at its bound or with its step)
  and when it ended.
- **Given** the command that launches runs under a host session that exports
  its session id, **when** the entry is written, **then** it names that
  session by id and by its persona name; **given** no session id is exported
  but the launching process descends from a host a session marker names,
  **then** it names that session; and **given** neither (a drain or build
  driven from a plain terminal), **then** it names the terminal run by its
  run id as the orchestrator, so every entry has one orchestrator and none
  is unattributed.
- **Given** the ledger, **when** any entry is written, **then** it is written
  under abcd's home in the machine-scoped store, one file per launch, and
  never in a worktree's local tier or the repository, so removing a worktree
  or a checkout deletes no entry and two launches never write one file.
- **Given** an entry, **when** a later abcd reads it, **then** a reader
  takes every field it knows and passes over those it does not, refuses
  nothing for a version above its own beyond naming it as newer in its
  explanation, and never writes an entry back; only the process that
  launched completes its own entry, and pruning removes whole files by age
  and count, so no reader on an older build is ever locked out as the run
  state's strict decode locks out a driver (iss-2611020719494343).
- **Given** a launch abcd handed a named key, **when** its entry is written,
  **then** the route reads key with the key's configured name, never its
  value, and its cost counts as spend; **given** a launch whose event stream
  reports the subscription's rate-limit windows, **then** the route reads
  subscription and its cost is kept as an estimate that counts in no sum of
  money; and **given** a launch abcd can place in neither, **then** the route
  reads unknown and its cost is likewise excluded, and the explanation says
  why.
- **Given** a launch killed at its time bound or with its step, **when** its
  entry is completed, **then** it carries what the transcript shows was
  spent, as the fallback receipt already keeps it, and the outcome stopped
  with the reason; **given** a launch whose harness process is gone with no
  end recorded, **when** the ledger is read, **then** the entry reads as
  lost, never as running.
- **Given** a session's entries, **when** the status line renders for that
  session, **then** its count of agents running and started, its count of
  failed or stopped agents, its spend per key against the key's budget, its
  burn rate since the session began and its agents' token total come from
  the ledger alone, read in one bounded pass that touches no network and
  finishes within the status command's deadline, with the result cached
  across refreshes for a short, configurable interval; and **when** a reader
  asks the explanation, **then** each of those figures names the ledger as
  its source, the token basis it counts, and the threshold it warns at.
- **Given** a session's entries, **when** the token total is shown, **then**
  the basis is the shipped default (every count the harness reports: prompt,
  completion, cache read and cache write) or the alternative the person
  configures (prompt and completion alone), and the explanation names which.
- **Given** the agents' API time, **when** the status line renders, **then**
  the time beside the session's name is the session's own, from the payload,
  and the agents' API time is summed in the ledger and shown by the verb and
  the explanation, not on the row.
- **Given** a person or an agent asks, **when** they run the ledger's verb
  bare, **then** it lists this session's launches (orchestrator, work, route,
  model, spend, tokens, API time, outcome, running or not) with totals, in
  text and `--json`; **when** they pass a run id, **then** it lists that
  run's launches and totals; **when** they pass a session id or ask for
  every session, **then** it lists those; and the verb reads only, writes
  nothing, and names an entry it could not read rather than dropping it
  silently.
- **Given** a run driven to done unattended (a drain from a plain terminal,
  or the build drive iss-2611031146255825 asks for), **when** it ends or is
  stopped, **then** its summary carries the ledger's totals for that run
  (launches, failed or stopped, key spend, tokens, API time) and names the
  verb that lists each launch, so the run's account of itself is the ledger
  and not a tally the driver kept.
- **Given** two entries for one await of one lane (a second implementer
  launched while the first's entry still runs or finished unreadable),
  **when** the ledger is read, **then** both are listed against the same
  await, the verb names the pair, and the status line's explanation counts
  it as a warning.
- **Given** the ledger's bounds, **when** entries pass the retention the
  person configures (a number of days and a count, with shipped defaults),
  **then** the next writer prunes the oldest whole files past them and the
  ledger never grows without bound; a pruned launch is gone from the ledger
  and from nothing else.
- **Given** the run state, **when** this intent ships, **then** its schema
  version is unchanged: the ledger names the run, lane and await, and the
  run state names nothing of the ledger, so no driver on an older build
  refuses a step because of it.

## Decisions

Ruled by the person on 2026-11-03, in the lab (lab-261103114820-bcba06f),
restated here; the draft is revised to them:

1. **Spend is split by billing route.** Spend and its budget sum the
   key-billed launches, and the session itself only when it bills a key;
   subscription work shows in the subscription's rate-limit gauges, which the
   host reports for the whole account and which therefore already include
   subscription agents; API time and tokens sum everything. A subscription
   session's cost figure is what the work would have cost and is never added
   to the bill (finding F-12).
2. **The row shows the session's own API time; the agents' time is recorded,
   not shown.** The person still wants each agent's API time kept per
   session, which has nowhere to live until the runner stores it per launch
   (F-13). The ledger is that place.
3. **The ledger, not the host, counts background agents.** The host's cost
   and token figures cover none of the agents a session starts in background
   shells, so only a ledger abcd keeps can count them (F-4). Agents a session
   starts by hand are outside it (scope conditions); the person's hand-kept
   spend log is the evidence for the per-key element, not a source the
   product reads.
4. **A token figure states its basis.** Cache reads dominate (F-5); the
   figure shown names what it counts, and the explanation repeats it.
5. **Every warning has a deterministic, readable explanation**, in the
   product a machine-readable form of the status line's state rather than a
   file, so a person, an agent and a test get the same answer (F-37). The
   ledger's figures carry their source, basis and threshold into it.
6. **The subscription's burn is a projection, the keys' is spend per hour**
   (F-17): the ledger supplies the key-billed spend over the session's
   elapsed time; the subscription's projection reads the host's gauges, not
   the ledger.

Settled by the record, so no question was put (2026-11-03):

7. **The write sits in the dispatcher.** Every launch abcd makes passes
   through the runner's dispatcher (the build's and drain's lanes, the drain
   judge and the planning interviewer each construct one), so the entry is
   written there, before the harness starts and again when it ends, and no
   launcher can forget it. The adapters read the harness's API time from the
   result event they already meter.
8. **The ledger is machine-scoped, under abcd's home**, beside the session
   markers, the transcript store and the worktree store, never in a
   worktree's tier (which `git worktree remove` deletes, and which a
   verifier's copy would read as the worktree's) and never in the
   repository. One file per launch, written by its launcher alone, so
   concurrent lanes and peers never contend for one file.
9. **The ledger adds nothing to the run state.** The run state's strict
   decode and its single version have locked a driver out three times
   (iss-2611020719494343); a per-launch record that bumped it would do so
   again, and its readers (the status line on the installed release, a peer
   on another build) are more often behind the writer than the run's driver
   is. The ledger has its own version, read leniently and never rewritten
   by a reader (the fourth criterion).
10. **The orchestrating session is the one the host exports.** The command
    that launches already reads the host's session id from its environment
    for the checklist's `updated_by`, and the session-start hook already
    keys a marker on the host process; the ledger uses the first, falls back
    to the second through the process walk the session register makes, and
    names the terminal run otherwise.
11. **Billing route from what abcd handed and what the stream shows.** abcd
    never reads a key's value and never asks an account; a named key it
    handed (itd-2611011230226453) is the key route, and a stream carrying the
    subscription's rate-limit events is the subscription's (the claude
    runner's stream shows them only on a subscription; a key-billed launch
    gets none). Neither is unknown, counted in no sum of money.
12. **Running is judged on the process, not a flag.** Each entry carries
    the harness process's id and start time, as a session marker carries
    the host's, so a reader checks liveness itself and an entry whose
    process is gone with no end written reads as lost.
13. **Impact: additive.** A new record and a new reader; nothing existing
    changes shape.

Settled at the planning interview of 2026-11-04 by the product thinker's
standing stance that abcd ships a sensible default and a person configures it
per repository (dcn-2611041550129278); each takes the draft's proposed
default, and none was put as a question of its own:

14. **Retention: 30 days and 20,000 entries**, pruned by the next writer; a
    machine or repository setting overrides both figures, with one floor: the
    retention is never shorter than the longest cap period of any key the
    budget intent keeps (itd-2611011230226453) plus one day, so a calendar
    month's totals are summed from entries still held, and a repository
    setting below that floor is read as the floor and named as such.
15. **The status line's cached read: 30 seconds**, the interval the
    prototype used; configurable the same way.
16. **Token basis: every count the harness reports**, as the prototype
    summed them; prompt and completion alone as the configurable
    alternative. The row shows one total naming its basis, and showing cache
    reads apart is a setting, off by default.
17. **The verb is `abcd agents`**, bare for this session, `--run <id>`,
    `--session <id>`, `--all`, `--json`; `launch` is the release verb and
    `launches` would read as its plural. The naming rule is applied at the
    verb audit as for any verb.
18. **The budget intent's per-key totals are summed from this ledger**
    (itd-2611011230226453), not kept in a store of their own, so one record
    holds every launch's spend; decision 14's floor is what makes the sum
    whole for the longest period a cap counts.
19. **A session's by-hand launches** (an agent started from a shell) are
    entered from their result file by a later intent, not this one.
20. **"Today" on the row is the machine's local day**, configurable to UTC.
21. **A role handed to the host session gets a placeholder entry** naming
    the host as the route, with no figures, so a run's listing is complete.

## Open Questions

_None open. The eight questions this draft carried are decisions 14 to 21,
each settled on its proposed default by the standing stance recorded in
dcn-2611041550129278._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

_Proposed for the product thinker to confirm at the planning interview and
record through `abcd intent ready --grounds`, not yet recorded: pursued
because abcd is about to leave runs working alone on a person's key, and we
expect one per-launch record written where every launch starts to make the
bill, the count and the account of an unattended run exact where the host's
counters and the run states cannot; we are wrong if sessions keep tallying by
hand, or if the ledger's running count and spend keep disagreeing with what
the person sees on their provider's console._
