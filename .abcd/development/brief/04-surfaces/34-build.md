# `/abcd:build` — Take One Intent From READY Towards Delivered

`/abcd:build` is the verb a person types to have abcd build one intent with
nobody holding the run in their head (itd-2609201916151817,
spc-2609202134338445). It starts the implement loop: a loop over a state file,
not a model, where every invocation reads the state, does at most one step,
writes the state and exits. `build` is the person's word and `implement` is the
machinery's (decision 8): the steps after the start are driven through the
[`/abcd:implement`](27-implement.md) family.

This chapter describes the part of the loop that ships: the checks, the pace
(itd-2609201925079472, spc-2609202134341288), the state file, the step
interface a host session drives with its window clock, and the lane's first
three steps (its worktree, its brief and the implementer's receipt). The validators
and the landing are named in the sequence and delivered by later pieces of the
spec; until each lands, the loop refuses at it by name.

## Sub-verbs

> _Machine-checked (`surface_coverage`, spc-27): each row records the verb's
> adr-40 bucket (`lint` / `review` / `audit` / `gate`, or `—` for a
> non-assessment verb) and its existence (`shipped` / `staged`). The existence
> fact is verified against the committed command-tree snapshot in both
> directions. The bucket cell is checked for membership of the closed adr-40
> vocabulary only: the snapshot carries no bucket field, so a bucket that is
> wrong but legal passes, and that cell stays a review-grain claim._

| Verb | Bucket | Status |
|---|---|---|
| `next` | — | shipped |

## The checks

No run is created until every check passes, and each is a read (criteria 1 and 2):

- **key** — the record is an intent. The issue key (decision 10) is refused by
  name until the piece that admits it lands.
- **ready** — the implement-readiness gate the intent verb reports: planned,
  criteria written, the spec linked both ways and written past its stub. Its
  advisory rows stay advisory.
- **open questions** — no open question under the record's `## Open Questions`.
  The reader fails closed and knows the record's two settled markers: a section
  that opens with an italic `_All resolved …_` line (or `_All four resolved …_`)
  is settled whole, and an item explicitly marked resolved or deferred — a bold
  span opening with the word (`**Resolved — …**`, `**Deferred**`, `**explicitly
  deferred**`, `**explicit deferral**`) or the word as a label (`resolved:`,
  `Deferred:`) opening a line of the item (a nested sub-bullet included),
  after a closing bold (with or without a colon after it) or after a dash — is
  not a question; the same word and colon mid-sentence are prose.
  Every other list item is a question whatever it says: one led `**Open`, one
  that only points to another record, and one that merely mentions deferral
  all count (the 2026-09-25 entry in `.abcd/work/DECISIONS.md`, and its
  2026-09-28 correction).
- **claim sections** — the mechanism prompt is answered or the section absent,
  and the scope conditions are recorded. The readiness gate reports both as
  advisory; a run is where they bind, because an autonomous lane has nobody to
  ask what the record meant.
- **hold** — the record carries no `held:`, well formed or not
  (iss-2609200830076665).
- **blocked** — nothing the record names in `blocked_by` is unshipped: an
  intent outside `shipped/`, or one this checkout's store does not hold, blocks
  it (itd-2609211116005482).
- **steps** — the open spec's `## Steps`, read through the spec store's own
  reader, parses and leaves at least one step unlanded. A spec listing no steps
  is one step, the whole spec.
- **peers** — no peer holds the intent: no sibling worktree or local branch
  holds it in a bucket other than this checkout's (a lane that shipped or
  re-drafted it), read through the peer listing, and no session other than the
  one the build is started for holds a live claim on it in the shared run
  state. A copy in the same bucket is not a
  holding: every branch cut from the default branch carries one. The check
  fails closed on what it cannot see into, on both sides: a peer the listing
  names and cannot read (git or the filesystem will not answer for it, or its
  ledger holds one id in two folders) and an unreadable claim file each count
  as a holding, naming why. A peer of another repository, or one holding no
  records at the committed layout, holds nothing of this checkout's and does
  not count.

A build started for a named session, one joined to the shared run, claims its intent in the shared run state when it creates the run, with
the run id as the lane and the longest lease a claim takes, so a build of the
same intent from any other checkout of the repository — another worktree or a
second clone on the machine — meets the claim at its peers check before this
run's lane has moved or claimed anything (iss-2609252050506863). The session's
own claim on the intent is its own, not a peer's. A session the shared run does
not hold is refused at the `claim` step before anything is created; a claim
refused under the lock leaves no run behind; a run whose state cannot be written
releases the claim it took. A build started without a session holds no claim
and says so, in the text and as a null `claim` in the JSON: until its lane
shows, another checkout cannot see it.

A refusal names the check, the reason and the remedy, carries every check's row,
and writes nothing. A peer's holding is contention rather than a fault in the
record.

## The pick

The pick sub-verb chooses the intent instead of taking one named
(itd-2609211116005482, spc-2609212015048113). Its candidates are the planned
intents that pass every check above, judged by the same function, less one
this checkout already has a run in progress for; a planned intent a check
excludes is named with the first check that excluded it. Each candidate is
scored from its record, three parts of 0 to 100 at equal weight (decision 7,
the bundled default): criteria clarity (the share of its acceptance criteria
with all three Given-When-Then clauses), a test path (its spec's `## Footprint`
section names tests) and the expected footprint (100 divided by the packages
that section names). A spec with no footprint scores zero on both and the
reason says so; the planning verb mints every spec with the section empty. The
readiest is taken and the oldest among equals (decision 2); `intent.PickLess`
is the order's one statement.

The pick starts the run `abcd build <itd-N>` starts for that intent, with the
pick in the state file (schema version 3). The reason is computed, never
composed (decision 5): one `pursued:` grounds entry whose text opens `picked by
run <run-id> on <date>`, then every candidate with its score, the rule, the
runner-up and why it lost, and the falsifier. The first lane's worktree step
appends it to the intent in the lane's own worktree, through the intent store's
grounds writer and lock, and commits that one file as the lane branch's first
commit (decision 6), under the configured git identity, with hooks off and the
isolated environment less the global-config neutralisers. A worktree step run
again adopts a commit already on the branch only when it is that commit byte
for byte: the pick's subject, the picked intent's record the one path changed,
and that record the base's with the one entry appended. The lane records the
commit as `pick_sha`, and the receipt verifier refuses a receipt naming it (it
is not the implementer's work) and a receipt over a branch that no longer
carries it, since a rebase or an amend that drops it drops the reason. The
readiness gate's grounds row skips a run-marked entry when it names the most
recent conjecture, so the person's entry stays the one it reports.

One pick per invocation. A run count above one, and a run until no candidate
is left, which continue under the pace rule (criterion 5), are refused by name; so is a pick whose
intent already has a run in progress. Naming a falsified pick in the run record
(criterion 6) waits on the unachievable verdict of itd-50.

## The pace

A run is paced without being told: a working window, a pause after it, and a
ceiling on the run's lanes and validators alive at once. The three numbers are
resolved once, when a new run is created, through the one layered configuration
reader (`internal/core/layered`), each key on its own, highest layer first:
the build's own pace and sub-agent flags, the pace written as
`<work-minutes>/<pause-minutes>`; `pace.work_minutes`, `pace.pause_minutes` and `pace.sub_agents` in the
repository's `.abcd/config.json`; the same keys in `~/.abcd/config.json`; and
the bundled default, 120 minutes of work, 300 of pause and 2 sub-agents
(decision 5), held in one set of constants. The files are read through the
reader's guards (a regular file inside the checkout; on the machine, one the
caller owns and nobody else can write), and the reader claims the `pace`
namespace, so a key under it the loop does not read is refused rather than
ignored.

The resolved pace is written into the run's state with each number's layer
(`flag`, `repo`, `machine` or `bundled`) and origin (the flag as typed, or the
file), the build's result carries it, and the run record's `pace` line names
it. A later invocation honours the pace the run started on, whatever the files
say by then; starting again with a flag naming another pace is refused, and one
naming the same pace resumes.

A malformed pace or ceiling is refused at the `pace` step naming the value and
the accepted form, and nothing is written (criterion 9): the pace flag is two
runs of digits around one slash, the work window 1 to 10080 minutes and the pause 0
to 10080, and the ceiling a whole number from 1 to 64; a configured value is
held to the same ranges, and one that does not decode as a whole number (a
string, a fraction, a null) is refused naming its file. A week bounds the
minutes so the window arithmetic stays far inside the clock's range and a typed
extra digit is refused rather than run. A pause of 0 minutes is a run that does
not pause.

The ceiling is recorded with the run; this build does not count lanes against
it (criterion 6), and the budget check and the rate-limit checkpoint (criteria
7 and 8) wait on a runner that reports its quota.

## The state file

A run lives in the checkout's local tier, one directory per run:

```text
.abcd/.work.local/run/<run-id>/state.json   the run
.abcd/.work.local/run/.lock                 the advisory lock every mutation takes
```

The run id is minted through the record-id seam, so two checkouts starting runs
in one second draw distinct ids. The tier itself is never created: only a
repository abcd manages has one, so a run is managed-only by construction. Each
run directory is created one level at a time and proved real, the state file is
replaced atomically inside an `os.Root`, and the reader decodes strictly,
refusing an unknown field, a schema version it does not know, or a file stored
under a run id it does not name. The state is schema version 2, which added the
pace. Version 1 is its strict subset, so a version-1 file is read as a run
started before the loop paced a run: it carries no pace, runs unpaced, and is
written back at version 2 by its next mutation. A version-1 file carrying a
pace is not one version 1 wrote, and is refused. A state file or run directory that is a symlink, or that
the filesystem will not hand over, is refused in the same shape (exit 2, naming
the file and the remedy), never followed.

The state holds the run's key, intent, spec and driver (the host session, by
default); the run's pace; the window clock (`window_started_at`,
`next_eligible_at`); the lanes opened so far; the spec steps still pending; and
the run record, one line per completed step and one per lane opened, naming the
spec step it builds (the start's line names the first). A lane carries its spec step and
title, its next step, what it awaits when a step has handed work to an agent,
and the footprint its steps fill in: branch, base and head, worktree, brief,
receipt and pull request. The status render names the worktree home-relative,
or by its directory name outside HOME (iss-2609281329007423); the brief and the
receipt stay whole paths, home-redacted, because the agent reads the one and
writes the other.

Starting creates one lane, for the first unlanded spec step, and records the
rest as pending. Starting again while that run is in progress creates nothing
and names the run. The live run for the key is looked up first, under the lock,
and the checks run only when a run is created: they judged the record at the
start, and the run's own lanes then change what they read (a lane's worktree
moves the intent to `shipped/`, a lane claims it), so judging again would refuse
the run as its own peer. Only the key's shape is checked before the lookup.

## The step interface

A host session drives the loop one step at a time (decision 5's default). The
lane's steps run in a fixed sequence: the worktree, the brief, the implementer,
the validators, the landing. Each invocation takes the lock, reads the state,
performs the current lane's next step and writes the state once, after the step
succeeds. A step that fails, or a process killed inside one, leaves the state as
it was, so the next invocation performs that step again; a step the state
records as done is never performed twice (criterion 7). A step's body is
therefore written to find what it made last time.

A step that hands work to an agent does not complete by itself: the lane then
awaits, naming the agent's role, the brief it is handed and the path its receipt
goes to (criterion 8). Asking again re-tells the same thing and moves nothing,
and the lane advances only when that receipt is handed back at that path and its
verifier accepts it. When a lane is done, the next pending spec step opens the
next lane, so the spec's steps land one lane at a time.

The loop keeps the run's window clock (criteria 4 and 5). A new run's first
window opens at its start. Once the window's working minutes have elapsed, a
step starts nothing: it writes `next_eligible_at`, now plus the run's pause,
records the pause, and exits 0 naming the time. The pause runs from the moment
the loop closes the window, not from the window's nominal end, so an invocation
that comes late never shortens it. An agent the lane already started may still
hand its receipt back during the pause, so the running lane finishes its step
and checkpoints to its branch; the receipt is not gated by the clock. Before
`next_eligible_at` a step is refused as a pause, naming the time, and nothing
moves (decision 1: no process sleeps through it); at or after it the next step
opens a new window, which the record names, and proceeds. A complete run is
reported complete and closes no window.

A step whose body this build does not carry is refused naming the step, the lane
and the spec piece that delivers it, and the run is unchanged, ready to resume in
a build that carries it. This build carries the worktree, the brief and the
implement step with its receipt's verifier; the validate and land steps are
refused naming pieces 8 and 9. The process driver (piece 3) is the same loop
called by a process instead of a host, starting the named agent through the
runner and handing its receipt back.

## The lane

A lane lands one step of the spec, and its files live in its own directory of
the run, `.abcd/.work.local/run/<run-id>/<lane-id>/`.

**The worktree** (piece 6). The lane's checkout is made in abcd's form: in the
machine-scoped worktree store, `~/.abcd/worktrees/<root-sha>/<run-id>-<lane-id>`,
keyed on the repository's root commit in the full form the sibling stores use,
on a branch `build/<run-id>-<lane-id>` cut from the default branch (origin's
`HEAD` as last fetched, else the first of `main`, `master`, `trunk` and
`develop`), never from whatever the checkout has checked out. The store's own
verbs are a draft (itd-2609091014076309), so the lane is a plain
`git worktree add` into the store's path. The path is derived, never taken: the
run id and the lane id are each held to their own shape, and the name they
compose must be one path segment of letters, digits, `.`, `_` and `-`, not led
by `-` or `.` and holding no `..`, so no component can leave the store. Every
level of the store is made one at a time and proved a real directory that is
the caller's alone (owned by the caller, writable by neither its group nor
anyone else), and a level the step makes is made `0700`, so a symlink anywhere
in the chain, or a level another account owns or can write, refuses the step
before anything is made inside it; nothing beside the checkout, and nothing outside
`~/.abcd/worktrees/<root-sha>/`, is created. Git runs in the isolated
environment, with `--` before the path. Run again after a kill, the step finds
the worktree git lists at the lane's path on the lane's branch and adopts it;
anything else at that path is refused and left as it is.

**The brief** (piece 5; criterion 3). The brief is rendered from the lane's
base commit, read out of git's objects rather than the worktree's files, so the
implementer reads the record its branch builds on, and a brief rendered again
after the implementer edited, committed or removed a file in the worktree is
still the base's: the intent and the spec whole, the conventions of `AGENTS.md` (its
section between `<!-- working-conventions … -->` and
`<!-- /working-conventions -->` when it marks one, the whole file when it does
not), and the decisions the intent cites (each ADR id in the intent, with its
title and path, and every entry of `.abcd/work/DECISIONS.md` that names the
intent or its spec). It opens by naming each source and the base it was read
at, then gives the lane (the spec step it builds, the worktree, the branch), the
spec's steps before the lane's (itd-2609212103565953, criterion 4), each with
what landed it — the `landed:` line the spec at the base records, and the lane
of this run that built it, with its branch, head and pull request — or that no
step comes before it, and
what the implementer hands back: its report, the definition of done's output,
and the receipt with its exact shape, each at an absolute path in the lane's
directory. Before the record it carries, the brief states the outbound policy
in its own right, quoted from `scanner.OutboundPolicy`, the value the lint
rules and the commit gates quote (itd-152): no live session URL and no tool
attribution footer in a pull request, an issue, a comment, a commit message or
a release note, and a re-read-and-strip of every pull request, issue and
comment the implementer creates, whatever the repository's own conventions
say. An intent the default branch does not carry as planned, a spec not
open there, or no `AGENTS.md` is refused rather than briefed from elsewhere, and
so is a source the base holds as a link or past its size cap, and a spec whose
steps the base cannot read, or lists the lane's step under another title than
the run opened it for: a run does not follow steps reordered mid-run, so a brief
naming the wrong predecessors is never written.
The brief is written atomically, mode `0600`.

**The receipt** (piece 7; criterion 4). The implement step hands the lane to a
fresh implementer and awaits its receipt at
`.abcd/.work.local/run/<run-id>/<lane-id>/receipt.json`. The receipt is the
implementer's word, read as untrusted input: through the guarded reader inside
the checkout's `os.Root` (no symlinked leaf, a regular file of at most 64 KiB),
decoded strictly (one JSON object, no field the schema does not name), with
every path it names held inside the lane's directory. Its fields are
`schema_version`, `run_id`, `lane`, `branch`, `commits` (full object names),
`definition_of_done` (`command`, `exit_code`, `output`), `report` and an
optional `model`, the model the implementer's harness reported. It verifies
only when every commit it names is on the lane's branch and not already on the
default branch at the lane's base, the definition of done's output exists
non-empty with exit code 0, and the report exists non-empty. A receipt short of
any of these is refused naming every gap at once, and the lane is not advanced.
A receipt carrying a verdict is refused by the same strict decode: a verdict is
the loop's to record (decision 9). A verified receipt moves the lane's head to
its branch's tip and the lane to its validators.

## Exit codes

`0` done, including a resumed start, a step that re-tells an await, a step
that closes an elapsed window, and a complete run; `2` refused, naming the step, the reason and the remedy, with
nothing written; `3` contention: a peer holds the intent, the run is paused, or
the run state is locked by another invocation. A refusal in the JSON form is its
own document before the error envelope, with the step, the check, the reason and
the remedy as fields.

## Where this sits

- The intent and its decisions: itd-2609201916151817; the design record:
  spc-2609202134338445, whose Progress section says which pieces have landed.
- The pace and the window clock: itd-2609201925079472 and its design record,
  spc-2609202134341288; the layered reader it resolves through is the model
  tier's (itd-2609170822093401).
- The worktree store the lane's checkout lives in: itd-2609091014076309 (a
  draft), and the rule it enacts, adr-2609091248200336.
- The shared run state and the claim the peers check reads:
  [`27-implement.md`](27-implement.md).
- The pick: itd-2609211116005482 and its design record, spc-2609212015048113.
- The plugin surface: `commands/build.md`.

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails `go test` when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd build`

Sub-verbs: `abcd build next`.

| Flag | Type |
|---|---|
| `--pace` | string |
| `--session` | string |
| `--sub-agents` | string |

### `abcd build next`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--max` | int |
| `--pace` | string |
| `--session` | string |
| `--sub-agents` | string |
| `--until-empty` | bool |

<!-- surface-appendix:end -->
