# `/abcd:build` — Take One Intent From READY Towards Delivered

`/abcd:build` is the verb a person types to have abcd build one intent with
nobody holding the run in their head (itd-2609201916151817,
spc-2609202134338445). It starts the implement loop: a loop over a state file,
not a model, where every invocation reads the state, does at most one step,
writes the state and exits. `build` is the person's word and `implement` is the
machinery's (decision 8): the steps after the start are driven through the
[`/abcd:implement`](27-implement.md) family.

This chapter describes the part of the loop that ships: the checks, the state
file and the step interface a host session drives. The lane's own steps, from
the worktree to the landing, are named in the sequence and delivered by the
later pieces of the spec; until each lands, the loop refuses at it by name.

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
  `Deferred:`) — is not a question. Every other list item is a question
  whatever it says: one led `**Open`, one that only points to another record,
  and one that merely mentions deferral all count (the 2026-09-25 entry in
  `.abcd/work/DECISIONS.md`).
- **claim sections** — the mechanism prompt is answered or the section absent,
  and the scope conditions are recorded. The readiness gate reports both as
  advisory; a run is where they bind, because an autonomous lane has nobody to
  ask what the record meant.
- **hold** — the record carries no `held:`, well formed or not
  (iss-2609200830076665).
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
and says so: until its lane shows, another checkout cannot see it.

A refusal names the check, the reason and the remedy, carries every check's row,
and writes nothing. A peer's holding is contention rather than a fault in the
record.

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
refusing an unknown field, another schema version, or a file stored under a run
id it does not name. A state file or run directory that is a symlink, or that
the filesystem will not hand over, is refused in the same shape (exit 2, naming
the file and the remedy), never followed.

The state holds the run's key, intent, spec and driver (the host session, by
default); the window clock the pacing intent writes (`window_started_at`,
`next_eligible_at`); the lanes opened so far; the spec steps still pending; and
the run record, one line per completed step. A lane carries its spec step and
title, its next step, what it awaits when a step has handed work to an agent,
and the footprint its steps fill in: branch, base and head, worktree, brief,
receipt and pull request.

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
next lane, so the spec's steps land one lane at a time. Before
`next_eligible_at` the loop refuses as a pause and nothing moves (decision 2).

A step whose body this build does not carry is refused naming the step, the lane
and the spec piece that delivers it, and the run is unchanged, ready to resume in
a build that carries it. The process driver (piece 3) is the same loop called by
a process instead of a host, starting the named agent through the runner and
handing its receipt back.

## Exit codes

`0` done, including a resumed start, a step that re-tells an await, and a
complete run; `2` refused, naming the step, the reason and the remedy, with
nothing written; `3` contention: a peer holds the intent, the run is paused, or
the run state is locked by another invocation. A refusal in the JSON form is its
own document before the error envelope, with the step, the check, the reason and
the remedy as fields.

## Where this sits

- The intent and its decisions: itd-2609201916151817; the design record:
  spc-2609202134338445, whose Progress section says which pieces have landed.
- The shared run state and the claim the peers check reads:
  [`27-implement.md`](27-implement.md).
- The plugin surface: `commands/build.md`.

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails `go test` when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd build`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--session` | string |

<!-- surface-appendix:end -->
