# `/abcd:build` — Take One Intent From READY Towards Delivered

`/abcd:build` is the verb a person types to have abcd build one intent with
nobody holding the run in their head (itd-2609201916151817,
spc-2609202134338445). It starts the implement loop: a loop over a state file,
not a model, where every invocation reads the state, does at most one stage,
writes the state and exits. `build` is the person's word and `implement` is the
machinery's (decision 8): the stages after the start are driven through the
[`/abcd:implement`](27-implement.md) family.

This chapter describes the part of the loop that ships: the checks, the pace
(itd-2609201925079472, spc-2609202134341288), the state file, the step
interface a host session drives with its window clock, the lane's five stages
(its worktree, its brief, the implementer's receipt, the validators and the
landing), and the run record read back at the end with the run's transcripts.

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

The status cell records that the sub-verb is in the command tree, not that its
intent is delivered: the pick is made once per invocation, and continuing under
the pace rule (criterion 5) and naming a falsified pick in the run record
(criterion 6) are not built ([The pick](#the-pick)).

## The checks

No run is created until every check passes, and each is a read (criteria 1 and 2):

- **key** — the record is an intent, or an issue id by shape (the issue key,
  decision 10). An issue takes two checks and no others: the repository's own
  drain rule takes it, read as the drain reads it ([`35-drain.md`](35-drain.md)),
  and no peer holds it out of `open/` or claims it. Its run has one lane, whose
  brief is the issue with its remedy as the work.
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
  it (itd-2609211116005482). A blocker in `superseded/` is followed along its
  `superseded_by` to the intent that replaced it, transitively, and the record
  waits on that replacement: it blocks exactly when the last intent of the
  chain has not shipped (ruling BZ2 of 2026-09-29). A chain that loops, names a
  record this checkout does not hold, stops at a superseded record naming no
  successor, or ends at a decision (`adr-N`) rather than an intent blocks, and
  the reason names the chain.
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
not hold is refused at the `claim` stage before anything is created; a claim
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
pick in the state file (schema version 3 added it). The reason is computed, never
composed (decision 5): one `pursued:` grounds entry whose text opens `picked by
run <run-id> on <date>`, then every candidate with its score, the rule, the
runner-up and why it lost, and the falsifier. The first lane's worktree stage
appends it to the intent in the lane's own worktree, through the intent store's
grounds writer and lock, and commits that one file as the lane branch's first
commit (decision 6), under the configured git identity, with hooks off and the
isolated environment less the global-config neutralisers. A worktree stage run
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

A run is paced without being told: a working window, a pause after it, a
ceiling on the run's lanes and validators alive at once, and the fix rounds a
lane may take before it is handed back (ruling DR1, 2026-09-29: a per-run value
set beside the pace, default 3). The four numbers are
resolved once, when a new run is created, through the one layered configuration
reader (`internal/core/layered`), each key on its own, highest layer first:
the build's own pace, sub-agent and fix-round flags, the pace written as
`<work-minutes>/<pause-minutes>`; `pace.work_minutes`, `pace.pause_minutes`, `pace.sub_agents` and
`pace.fix_rounds` in the
repository's `.abcd/config.json`; the same keys in `~/.abcd/config.json`; and
the bundled default, 120 minutes of work, 300 of pause, 2 sub-agents and 3 fix
rounds (decision 5 and ruling DR1), held in one set of constants. The files are read through the
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

A malformed pace or ceiling is refused at the `pace` stage naming the value and
the accepted form, and nothing is written (criterion 9): the pace flag is two
runs of digits around one slash, the work window 1 to 10080 minutes and the pause 0
to 10080, the ceiling a whole number from 1 to 64, and the fix rounds a whole
number from 0 to 64 (0 hands a lane back on its first round that does not
pass); a configured value is
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
under a run id it does not name. The state is schema version 7. Version 7
added the landing (a lane's `landing`, the implementers' `receipts` it verified
with the model each runner reported, and the captures its receipts declared
fixed, `resolves`) and the run's captured `transcripts`. Version 6
added the fix-round cap (ruling DR1): the pace's `fix_rounds` and a lane's
`hand_back`. Version 5 added the validate stage's record (a lane's
`validation`). Each earlier version is the next one's strict subset, read as a
run that predates the addition (a version-5 run runs on the bundled cap) and
written back at version 7 by its next mutation; an earlier version carrying what
only a later one writes is refused. Version 4
renamed the lane's stage (BU1, iss-2609291313276243): a lane's and a record
line's `step` became `stage`, so "step" names only the spec's steps (`spec_step`,
`step_title`, `pending`). Versions 1 to 3 wrote `step`, and are migrated on
read: the read carries each `step` over to `stage` and writes nothing, and the
run's next mutation writes the file back at version 4; one of them that already
says `stage` is not one its version wrote, and is refused. Version 3 added the
pick and version 2 the pace. Version 1 is version 2's strict subset, so a version-1 file is read as a run
started before the loop paced a run: it carries no pace, runs unpaced, and is
written back at version 2 by its next mutation. A version-1 file carrying a
pace is not one version 1 wrote, and is refused. A state file or run directory that is a symlink, or that
the filesystem will not hand over, is refused in the same shape (exit 2, naming
the file and the remedy), never followed.

The state holds the run's key, intent, spec and driver (the host session, by
default); the run's pace; the window clock (`window_started_at`,
`next_eligible_at`); the lanes opened so far; the spec steps still pending; and
the run record, one line per completed stage and one per lane opened, each
naming its `stage`, the lane-opening line naming the spec step the lane builds
(the start's line names the first). A lane carries its spec step and title
(`spec_step`, `step_title`), its next `stage`, what it awaits when a stage has
handed work to an agent, and the footprint its stages fill in: branch, base and head, worktree, brief,
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

A spec's steps and a lane's stages are two words for two things (BU1,
iss-2609291313276243): each spec step lands as one lane, and the loop takes the
lane through its stages. The step verb performs one stage.

A host session drives the loop one stage at a time (decision 5's default). The
lane's stages run in a fixed sequence: the worktree, the brief, the implementer,
the validators, the landing. Each invocation takes the lock, reads the state,
performs the current lane's next stage and writes the state once, after the
stage succeeds. A stage that fails, or a process killed inside one, leaves the
state as it was, so the next invocation performs that stage again; a stage the
state records as done is never performed twice (criterion 7). A stage's body is
therefore written to find what it made last time. The result names the stage
the call completed as `performed_stage` and the lane's next as `stage`.

A stage that hands work to an agent does not complete by itself: the lane then
awaits, naming the agent's role, the brief it is handed and the path its receipt
goes to (criterion 8). Asking again re-tells the same thing and moves nothing,
and the lane advances only when that receipt is handed back at that path and its
verifier accepts it. When a lane is done, the next pending spec step opens the
next lane, so the spec's steps land one lane at a time.

The loop keeps the run's window clock (criteria 4 and 5). A new run's first
window opens at its start. Once the window's working minutes have elapsed, the
step verb starts nothing: it writes `next_eligible_at`, now plus the run's pause,
records the pause, and exits 0 naming the time. The pause runs from the moment
the loop closes the window, not from the window's nominal end, so an invocation
that comes late never shortens it. An agent the lane already started may still
hand its receipt back during the pause, so the running lane finishes its stage
and checkpoints to its branch; the receipt is not gated by the clock. Before
`next_eligible_at` the step verb is refused as a pause, naming the time, and
nothing moves (decision 1: no process sleeps through it); at or after it the
next stage opens a new window, which the record names, and proceeds. A complete run is
reported complete and closes no window.

A stage whose body this build does not carry is refused naming the stage, the
lane and the spec piece that delivers it, and the run is unchanged, ready to
resume in a build that carries it. This build carries every stage of the
sequence. The process driver (piece 3) is the same loop
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
anyone else), and a level the stage makes is made `0700`, so a symlink anywhere
in the chain, or a level another account owns or can write, refuses the stage
before anything is made inside it; nothing beside the checkout, and nothing outside
`~/.abcd/worktrees/<root-sha>/`, is created. Git runs in the isolated
environment, with `--` before the path. Run again after a kill, the stage finds
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

**The receipt** (piece 7; criterion 4). The implement stage hands the lane to a
fresh implementer and awaits its receipt at
`.abcd/.work.local/run/<run-id>/<lane-id>/receipt.json`. The receipt is the
implementer's word, read as untrusted input: through the guarded reader inside
the checkout's `os.Root` (no symlinked leaf, a regular file of at most 64 KiB),
decoded strictly (one JSON object, no field the schema does not name), with
every path it names held inside the lane's directory. Its fields are
`schema_version`, `run_id`, `lane`, `branch`, `commits` (full object names),
`definition_of_done` (`command`, `exit_code`, `output`), `report`, an
optional `model`, the model the implementer's harness reported, and an optional
`resolves`: each capture the lane fixed, with the `commit` that fixed it, the
`note`, the `impact` and the `grounds` its resolution records. It verifies
only when every commit it names is on the lane's branch and not already on the
default branch at the lane's base, the definition of done's output exists
non-empty with exit code 0, the report exists non-empty, and each fixed capture
is an issue id named once, with one of the receipt's own commits, an impact from
the changelog's enum and a note and grounds within their cap. A verified
receipt is recorded on the lane with its model, and its fixes with it, a later
fix round's declaration of an issue replacing an earlier one's. A receipt short of
any of these is refused naming every gap at once, and the lane is not advanced.
A receipt carrying a verdict is refused by the same strict decode: a verdict is
the loop's to record (decision 9). A verified receipt moves the lane's head to
its branch's tip and the lane to its validators.

**The validators** (piece 8; criteria 5 and 12). The validate stage hands the
lane's head to validators that did not implement it, one fresh agent at a
time, each with a brief the loop renders into
`.abcd/.work.local/run/<run-id>/<lane-id>/validate/round-<n>/<role>/`: a
`ruthless-reviewer`, then a `security-reviewer`, each over the lane's diff from
its base to its head, and, on the lane whose landing closes the spec, an
`intent-auditor`. The fidelity audit runs once, on that lane, over the whole
delivery (ruling AI, 2026-09-29): its request carries the range from the base of
the run's first lane to the closing lane's head, each lane's own range, and
every spec step landed before the run by what landed it. A lane that does not
close the spec takes no audit step, and neither does a closing lane whose close
leaves the intent planned because another open spec names it: the criteria are
the intent's, audited once, whole. The request is composed as the close's own
emit composes it, keyed on the receipt the close parks and written against the
path the close moves the intent to, so the auditor's verdict is the one the
landing's close consumes; the delivered range sits after its Provenance block,
outside the prompt hash. Only the loop writes a verdict (decision 9): each
validator writes its return, and the loop parses the verdict out of it — a
reviewer's one `### Verdict` section stating one verdict of its role (`SHIP` or
`FIX FIRST`; `APPROVE`, `BLOCK` or `NEEDS-INPUT`), the auditor's fidelity
verdict checked against the request (its receipt, both provenance hashes, every
criterion and scope condition) — and records it into the state file; a return
the loop cannot read one verdict from is refused and the lane still awaits it.
A round whose validators all pass completes the stage, unless a report the
lane's receipts name states a verdict (`Verdict: SHIP`, or a Verdict heading
over one), which is refused at the advance naming the report. A round one of
them did not pass (`FIX FIRST`, `BLOCK`, `NEEDS-INPUT`, a criterion `NOT_MET`)
goes to a fresh implementer, who applies each finding with a commit or rejects
it in writing in its report, and hands back a receipt verified as the
implementer's is; the next round then hands the lane's head to every validator
again, so no verdict stands over a head it did not read and a rejection is
judged by the validator it answers. The state records every round with the head
it judged, each verdict, and the audit's receipt and range.

The audit passes a round only when it judges every criterion met: a criterion
it could not decide (`INCONCLUSIVE`) fails the round exactly as a `NOT_MET` one
does, and the fix brief names it as undecided, so the lane never lands on an
audit that decided nothing (ruling DQ1a, 2026-09-29: an undecided audit reopens
the work, never closes like a pass). A return the loop cannot read as a verdict
records nothing and is refused, so it starts no fix round and counts against
nothing.

**The fix-round bound** (ruling DR1; itd-50, criterion 2). A round that does not
pass once the lane has taken the run's cap of fix rounds starts no further fix
round: the lane stops at the `handed-back` stage with a `hand_back` record (the
verdict `unachievable`, the round, the cap, the last round's verdicts, the
returns of the validators that did not pass, and the criteria the audit judged
not met or could not decide). The step's result carries it and says so first;
the run record gains a `handed-back` line, and, for a run the pick
started, a `pick` line naming the pick falsified (itd-2609211116005482), the
intent's grounds entry left as written. The run starts nothing further for the
lane: every later step is refused at the `handed-back` stage naming the
hand-back, and building the intent again resumes the run and says the same.
Moving the intent to `drafts/` with its replan reason (itd-50, criterion 3) is
not made by this build.

**The landing** (piece 9; criterion 6). The land stage takes the lane to the
default branch one step per invocation, each recorded in the lane's `landing`
as it completes, so a killed invocation repeats the step that did not complete
and finds what it made rather than making it twice. The lane stays at `land`
until the last step.

1. It checks the lane's worktree is clean and its branch is at the head the
   validators judged, and decides whether the landing closes the spec: the
   lane that took the fidelity audit does.
2. In the lane's worktree it runs the spec's close (which ships the intent
   and parks the fidelity receipt) and ingests the verdict of the audit
   the lane took into that receipt, rather than asking for a second audit; and
   for every capture the lane's receipts declared fixed it runs the capture
   store's resolve with the lane's commit that fixed it. It commits them on
   the lane's branch with a computed message carrying `Delivers:` (when the
   close ships the intent) and one `Resolves:` per capture, so RS005 and RS001
   find the records in the change. A lane that neither closes the spec nor
   fixed a capture records nothing.
3. It pushes the lane's branch to `origin` only once the repository's
   preflight receipt (`.abcd/.work.local/preflight-receipts/<head>`, in any
   worktree git lists) names the lane's head, the gate the pre-push hook
   checks, read before any connection opens. The push is a plain `git push`
   from the checkout the run lives in, so the hook runs; nothing is skipped or
   forced.
4. It opens the pull request through the forge client the repository already
   uses (`gh`), with a title and a body written from the run's records (the
   step, the spec, the intent, the passing round's verdicts, the close, each
   resolved capture, and the `Delivers:` and `Resolves:` lines) and passed
   through the outbound scrub. After creating it, the loop re-reads the body the
   forge holds, and a session URL or tool footer the harness appended is
   stripped and the body read again; one that survives is refused. A pull
   request a killed invocation opened is found by the forge's listing of the
   branch, never opened twice.
5. It reads the merge rule from the ruleset mirror (`.abcd/work/rulesets/`) at
   the lane's base, so the lane's own commits cannot change it (decision 3):
   where an active ruleset gates the default branch through a merge queue, it
   arms auto-merge with the queue's method; where none does, it leaves the pull
   request open for a person to merge. Nothing is pushed to the lane after this
   step.
6. It fetches the default branch and waits, exiting 3, until the pushed head
   is an ancestor of it; only then does it remove the lane's worktree (never
   forced) and delete the lane's branch at a tip the same check proves landed,
   and the lane is done. A pull request closed without merging, or merged in a
   way that rewrote the head, is refused and nothing is cleaned up.

**The run record and the transcripts** (piece 10; criterion 10). The record
verb reads a run's state back as its record: every lane with its spec step,
branch and heads, the implementers' receipts the loop verified with the model
each runner reported (as reported; the binary cannot verify it), every verdict
the loop recorded, round by round, the captures the lane fixed, its pull request
and what its landing did, the run's pending steps, the transcripts captured and
the record's lines, in text and JSON. On a complete run, the record verb
captures each transcript it is named into the history store as the history
verb's capture of one path does, one capture per path, and records it in the
state with the session it was stored under; a run in progress is refused, since
the record's transcripts are the run's, captured at its end. A capture that
fails stops the call, with the transcripts before it recorded.

## Exit codes

`0` done, including a resumed start, a stage that re-tells an await, a call
that closes an elapsed window, and a complete run; `2` refused, naming the stage, the reason and the remedy, with
nothing written; `3` contention: a peer holds the intent, the run is paused, the
run state is locked by another invocation, or a landing waits for its pull
request to merge. A refusal in the JSON form is its
own document before the error envelope, with the stage (`refusal.stage`), the check, the reason and
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
| `--fix-rounds` | string |
| `--pace` | string |
| `--session` | string |
| `--sub-agents` | string |

### `abcd build next`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--fix-rounds` | string |
| `--max` | int |
| `--pace` | string |
| `--session` | string |
| `--sub-agents` | string |
| `--until-empty` | bool |

<!-- surface-appendix:end -->
