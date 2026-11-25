---
name: lab
description: "List this repository's labs with their pins, probe counts and halts: Writes nothing; refuses outside a git checkout."
block: people
---

# `/abcd:lab`

<!-- generated: ask-the-binary -->
Before you fill in an input for this command, run `"${CLAUDE_PLUGIN_ROOT}/abcd" explain lab`, and where its answer and this page disagree, follow the binary, since this page may be older than it.
<!-- /generated -->

Run a lab against a pinned snapshot of this repository. The lab's evidence lives
in the machine-scoped lab store (`~/.abcd.noindex/lab/<root-sha>/<lab-id>/`), its
knowledge enters the record only through capture, and **no lab verb writes into
the repository**. The procedure the verbs encode is the discipline record
`itd-2609251624540864` in `.abcd/development/intents/disciplines/`.

Pick the sub-verb from the user's input; with none, list the labs:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" lab --json
```

The payload's `labs` each carry `id`, `pin`, `question`, `created`, `home`
(tilde form), `probes` and `halted` (the gates holding the lab halted). An
absent store is an empty list and is not created.

**Mint** a lab for one question, pinned at HEAD or at `--pin <commit>`:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" lab mint --json "<question>"
```

It creates the lab home and its registry line, clones a standalone snapshot
detached at the pin (no hook fires, its remote is cut) and scaffolds
`INTENTION.md`, `findings.md`, `corrections.md` and `amendments.md`. Show the
user the `id`, the `home` and the `next` steps: write the hypothesis and STOP
conditions before anything mutates, and, in a lab of abcd's own repository,
build the work binary once from the pristine snapshot into `bin/abcd`.

**Preflight** the lab before any mutation:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" lab preflight --json <lab-id>
```

`checks` holds the harness-isolation group (`isolation.home`,
`isolation.snapshot`, `isolation.remotes`, `isolation.hooks`) and the dual-binary
group (`binary.work`, `binary.pinned`, `binary.test`), each with `ok` and
`detail`; the artefact is `state/preflight.md`. The dual-binary group holds a
lab of abcd's own repository only — its root commit is abcd's, or the pin
commit carries `cmd/abcd/main.go`: in any other repository no `bin/abcd` can be built
from the snapshot, so each binary check passes with `not_applicable` true and a
`detail` saying why — tell the user it was judged not applicable,
never that it passed on its merits. On any failure `passed` is false,
the exit is 1, and `finding` names the gate finding the halt recorded. Tell the
user which checks failed and why. **Do not adapt around a failed check** — no
retry under another shape, no weakened setting: the lab is halted until the
preflight passes again, and the finding stays.

**Record** a probe before running it:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" lab record --json <lab-id> <probe-name>
```

It scaffolds `state/probes/<probe-name>/` — `input`, `argv`, `exit`, `stdout`,
`stderr` and `record.md` naming the artefact observed. The verb runs nothing:
run the probe's command yourself and redirect its output into those files as
`fill` shows. A halted lab refuses (exit 1); a probe already recorded is refused
(exit 2) — a re-run is a new probe.

**Sweep** the lab's retractions:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" lab sweep --json <lab-id>
```

Each `- retract: \`<literal>\`` line in `corrections.md` is searched for across the
lab's own documents; `corrections` lists each with `applied` and every
`instances` file and line. An unapplied correction fails the sweep (exit 1) and
halts the lab with a gate finding. `not_swept` lists every document the sweep
could not read (too large, binary, or not a regular file), by path; while any
correction is recorded, one such document fails the sweep too. Report every
instance and every document not swept; the fix is to remove the claim
everywhere it stands, and to make an unread document readable or move it out
of the lab home, then sweep again.

**Harvest** the lab:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" lab harvest --json <lab-id>
```

It writes `harvest/harvest.md` in the lifeboat's section shape, citing each
finding's probe records. `candidates` are the product findings, each with the
`command` that files it as a capture found during the lab — **file nothing
yourself**: show the candidates and let the user decide, and note any without a
`refutation`. When a finding cites no probe record or evidence, or one that is
missing or incomplete, `gaps` names it, the exit is 1 and nothing is written.

When the lab collected the decision adapter's shadow pairs, the harvest also
carries `agreement`. A judgement type is put in shadow by setting
`oracle.judgements.<type>` to `shadow:<provider>/<model>` in
`~/.abcd.noindex/config.json`. The host keeps deciding, and each answer the
adapter gives beside it is appended to `pairs.jsonl` in the newest lab home.
The adapter is the first candidate, a typed-decision model such as
`typesafe/jev-1.13`, unless the provider's block names the model in
`logprob_models`: then it is the second, a small text model whose options are
scored by the log-probability of their one-token labels. Each pair names the
`<provider>/<model>` that answered; the harvest counts by type, so compare the
two candidates in separate labs.
Two steps go through the router today, so theirs are the types whose pairs
are kept: the drain's judgement before a lane opens (`eligibility-residual`),
and embark's lesson ranking (`embark-lesson-ranking`), one pair per lesson once
the host's scores are handed back. Every other type stays on the host whatever
its route says (the drain and `embark lessons` each print a diagnostic for a
route set on such a type), and a lab with no pairs has no `agreement`.
For each type, `agreement` gives `pairs`, `compared` (the adapter answered),
`agreed`, `skipped` (the state was over the model's window, or the call
failed), `rate`, and the compared and agreed counts in four probability
`bands`, which the text prints on the line under the type's. It ends with the `note` line, naming the lab home and the count. Show
the user the rates and the note: a research note filed from it is the evidence
a ruling turns a type on with (`<provider>/<model>`, no `shadow:`), and turning
a type on is the user's decision, never yours.

**Plan a model matrix run** in a lab whose preflight passed:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" lab matrix --json <lab-id> <declaration.json>
```

It shows the plan and writes nothing: `cells` (each cell's role, model and
`source`: host, openrouter, local or provider), `copies` (the pilot and the most
the run may make), `cost` (the expected cost by source at the most copies,
reckoned only from each cell's declared `cost_per_copy`; a metered source with a
cell that declared none says `not estimated; bounded by your cap`), `spend_cap`
and where it came from, `load_cap` and `cores`, `time` (a bound when every cell
declares `minutes_per_copy`, otherwise known after the pilot), `judges` (each
with its `note`), `hidden_tests` (each code task's `folder`, its test `command`
word by word, and the `consent` line the person agrees to), and the `digest`
naming exactly this plan, with `approve_with`, the command that starts it.
Building it makes no network call and fetches no price. Show the person the
whole plan in plain words: the copies, what it may spend against the cap, the
load, the time and the judges, and each code task's hidden-tests folder and
command with its consent line, word for word. The digest covers them, so never
pass `--approve` for a plan whose hidden tests the person has not seen.

**Only after the person has seen that plan and said yes to it**, record the run
with the digest it carried:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" lab matrix --json <lab-id> <declaration.json> --approve <digest>
```

The plan is built again and must carry that digest; an approval naming another
plan is refused with nothing written. Never pass `--approve` on your own
judgement, and never with a digest the person did not see. The run then hands
off every pilot copy: `copies` lists each one's `key`, `state`, `clone` (its own
clone of the frozen commit under a random alias, outside every checkout) and
`brief` (what the copy is asked, naming no task id, cell, route, model or path
on this machine). A
clone bound for a provider passes the launch bundle's secret scan first, over
its checked-out tree and the frozen commit's own message, author and committer,
and a hit leaves the cell `refused`. A host cell's copy is `handed_off`: it is yours to
make, below. A runner cell's copy is `awaiting_runner`: abcd does not launch a
runner in a clone yet, and nothing stands in for it, so never make it yourself
and tell the person plainly that those cells wait. Refused, exit 2, with nothing
written: a metered cell with no spending
cap (set one in the declaration's `spend_cap` or `matrix.spend_cap` in the
person's own machine configuration, never the repository's), a local cell with
no load cap or one at or above the cores, a route this machine's runners refuse,
an OpenRouter or local cell with no pin, and a stale approval.

**Make each host cell's copy and hand it back.** Show a copy's hand-off with:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" lab matrix copy --json <lab-id> <run-id> <task>/<cell>/<repeat>
```

Hand its `brief` to a fresh sub-agent started in its `clone`, which the brief
calls the current directory, and working nowhere else,
telling it nothing more of the run. For a code copy, then run the task's hidden
tests against the clone yourself, through the load gate, outside the clone.
Hand the copy back with a receipt file, `{"schema_version": 1, "outcome":
"made"}` or `"failed"` with a `reason`, with `started_at`, `finished_at`,
`model_time_ms` and `gate_wait_ms` where known, and for a made code copy
`"score": {"passed": <n>, "total": <n>}` from those hidden tests:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" lab matrix copy --json <lab-id> <run-id> <key> --receipt <file> --transcript <file> [--answer <file>] [--form doc-fidelity|consistency]
```

`--transcript` is the sub-agent's transcript, which the compliance check reads
for reads outside the clone (without it that rule is unexamined and the copy is
reported apart). A writing copy names its document with `--answer`; a check
copy its answer with `--answer` and `--form`. The copy is checked for
compliance first, and the trial (`status`, `reason`, `score`, `compliance`) is
written once; the trial says the host, not abcd, ran the hidden tests, and its
`unpinned` marks the score, the model served, the harness version and the times
as `self_reported` where the receipt gives them and `trial.calls` as
`not_reported`, since a host copy carries no call receipts. A writing copy's
document is kept only once its trial passes the secret screen. A check
copy's clone is deleted only once its trial is written. Refused, exit 2, with
nothing written: a runner cell's copy, a copy already handed back, a receipt,
score or answer that does not validate, and a trial the secret screen refuses.
A refusal raised once a check copy's findings are recorded, or a writing copy's
document is kept, says so instead.
Hand back every host copy you were handed, a failed one with its reason: a copy
never handed back is in no rate, and the report names it.

**Review a code task's copies**, once every copy of it handed to you is back:
the review is drawn once, so it is refused while one is out.

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" lab matrix review --json <lab-id> <run-id> <task>
```

`requests` gives each reviewer cell's blinded `request` about each copy under
an opaque `label`. Hand each `handed_off` request, and nothing else, to a fresh
sub-agent, and save its one JSON answer at `<dir>/<reviewer-cell>/<label>.json`;
a request `awaiting_runner` waits, as a runner cell's copy does. Every request
passes the secret screen before it is stored or handed out, and a refusal
draws nothing. Then record
them, read as untrusted input and scored before anything is written:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" lab matrix review --json <lab-id> <run-id> <task> --answers <dir>
```

**Draw a writing task's judging** from its made copies, once every copy of it
handed to you is back, with one good and one bad example the person gives and
how many judges' subsets to draw:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" lab matrix judge --json <lab-id> <run-id> <task> --good <file> --bad <file> --judges <n>
```

The packet, written in the run's state with its key beside it, renders every
copy as inert text with every mention of where it ran, its alias, every task,
cell, route and model and every maker withheld, however it is spelt (split by
invisible characters or by separators between its letters, in look-alike
letters, or with its version after it), and names no run, task or path.
The packet and key pass the secret screen before they are written, and a
refusal draws nothing. It is what the person's judging session shows; hand the session's verdict file
back with `--verdicts <file>` (and `--host <login>` when the host judged too,
whose rankings are reported apart).

**Judge a check task** in two phases, once every copy of it handed to you is
back:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" lab matrix judge --json <lab-id> <run-id> <check-task>
```

It joins the copies' findings by their exact keys and shows the judging's
`stage`, its `copies` (a copy the compliance check set apart is in none),
`leftovers` (the findings no exact key joins) and the declaration's `matcher`
and `proposer` cells. With no matcher declared, the leftovers stay rows of their
own and the `packet` is drawn at once. With one, the verb first draws the
`matcher_request` about the leftovers alone (`awaiting_matcher`): hand it, and
nothing else, to a fresh sub-agent for a host matcher cell, save its one JSON
answer, `{"schema_version": 1, "same": [["<id>", "<id>"]]}`, and record it:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" lab matrix judge --json <lab-id> <run-id> <check-task> --matcher <file>
```

The answer is refused whole, with nothing recorded, when it names a finding it
was not shown; once taken, its pairs are joined, a `sample` of pairs is drawn
for the person to label, in an order that says nothing of which pairs the
matcher joined, and the `packet` is drawn. A matcher or proposer cell marked
`awaiting_runner` waits for a launcher this build does not have: the judging
stops at `awaiting_matcher` or `awaiting_proposer` with `waiting` saying so.
Never answer for it yourself, and tell the person it waits. Where `waiting` says
the draw of the packet was interrupted, run the verb with no flag: it draws the
packet from what the draw recorded. The packet shows each distinct finding under a `code`,
shuffled, with no model, cell, run or path named and no verdict proposed. Show
it to the person and record their verdict on every code, locked, with
`--verdicts <file>`: `{"schema_version": 1, "packet": "<packet_digest>",
"locked": true, "verdicts": [{"code": "<code>", "valid": true}]}`. The verdicts
lock once. Then build the second phase:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" lab matrix judge --json <lab-id> <run-id> <check-task> --table [--proposals <file>] [--labels <file>]
```

`--proposals` is the proposer cell's verdicts on the same packet, from a fresh
sub-agent shown the packet alone: `{"schema_version": 1, "packet":
"<packet_digest>", "proposed": [{"code": "<code>", "valid": true, "reason":
"<one line>"}]}`. It is required when the declaration names a host proposer and
refused when it names none, and its reasons pass the secret screen before the
table is written. `--labels` is the person's labels on the sample,
`{"schema_version": 1, "labels": [{"a": "<id>", "b": "<id>", "same": true}]}`.
The table gives each finding's row, each model's `scores` (precision, unique
valid findings, recall) and the `overlap`; `matcher` gives its agreement, or
`not_estimable` saying why: no matcher declared, none asked, or fewer than ten
pairs labelled. Asked again with no flag, the verb shows the judging as it
stands. Refused, exit 2, with no record written: a task with a copy handed to you
and not back, a matcher, verdict, proposal or label file that does not
validate, a second matcher answer, verdict file or table, a table before the
verdicts are locked, an answer for a runner cell, and a matcher or proposer
the run's record does not pin. A run that cannot be read is refused as it is,
whatever the task. Nothing of a check is
filed or written where the release gates read.

**Read a recorded run's report**, which writes nothing:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" lab matrix report --json <lab-id> <run-id>
```

`code` gives, per task, each builder cell's `pass_at_1` (passes of the attempts
that count, the rate and its 95% `interval`), `host_reported` (how many of
those rest on the host's own report of its hidden tests), `pass_hat_k`, the copies set
apart (`not_started`, `unexamined`, `excluded`, in no rate), `reviews`
(`confirmed` findings on failing copies, `unconfirmed` flags on passing ones),
`costs` by unit with `per_pass` and what the figure is `comparable` with, and
`model_time_ms` apart from `gate_wait_ms`; and `comparisons`, each `ranked` only
at the repeats its power estimate asks for (`needed`, from the declaration's
`repeats.min_difference`), otherwise stating `min_detectable`, the smallest
difference it could have detected, in its `note`; one resting on host-reported
results says so in its `note`, with `host_reported` set, and a limitation names
those copies. `writing` gives each model's
`ratings` (a regularised Bradley–Terry rating, its `interval` and whether it is
`separable`), `agreement` (alpha with an interval, or `not_estimable` saying
why), `good_enough` and each copy's length; with fewer than three judges,
`floor` says so and `rankings` holds each judge's ranking instead. `check` gives
the table the person's locked verdicts unlocked, null until then. `drift`, on a
re-run, lists every field that drifted from the first run with both values.
`unreturned` lists every copy handed off that holds no trial, by task, cell
and `state` (`not_handed_back`, `awaiting_runner`, `refused`, or
`never_handed_off` when the run's start stopped before it), with its
`repeats`; each code cell counts them too, and none is in any rate. Show the person the rows and read them `limitations` and `improvements` in
plain words: a comparison that is not ranked is not a result either way.

**Re-run a recorded run** from its own declaration, by its id in the lab or a
run's folder carried from elsewhere:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" lab matrix rerun --json <lab-id> <run-id|run-folder>
```

It shows the plan as the first form does and writes nothing; record the re-run
only after the person has seen that plan and said yes to it, with the digest it
carried (`--approve <digest>`). The re-run's record is written beside a copy of
the first run's records, its pilot copies are handed off as the first form's
are, and its report shows the drift. Refused, exit 2, with
nothing written: a run the lab never recorded, a folder holding no run's
records, a run whose frozen commit is not the lab's pin, a run whose records
the secret screen refuses, and a stale approval.

Exit codes: `0` done; `1` a gate refused or a halted lab refused a probe; `2`
the request was refused (no checkout, an unknown lab, a bad name, question or
pin), with nothing written.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout. In a checkout the
person has listed in `~/.abcd.noindex/dogfood-roots`, or a worktree that
checkout records, `"${CLAUDE_PLUGIN_ROOT}/abcd" hook dogfood-binary` prints
the path of a build of the reviewed branch's tip, as the remote that line
declares last reported it, once it is ready: run that path in place of the
plugin-root binary, and the plugin-root binary while it prints nothing.

<!-- generated: command-facts -->
**Command facts, format version 2.** Where this block and `"${CLAUDE_PLUGIN_ROOT}/abcd" explain lab` from the running binary disagree, follow the binary.
<!-- Written by `make command-facts-sync` from the command tree; change the command in internal/surface/cli or internal/core/surface, never this block. -->

- `abcd lab [flags]`
  - An example call is `abcd lab`.
- `abcd lab harvest <lab-id> [flags]`
  - An example call is `abcd lab harvest lab-260901000000-0123abc`.
- `abcd lab matrix <lab-id> <declaration.json> [flags]`
  - An example call is `abcd lab matrix lab-260901000000-0123abc matrix.json`.
- `abcd lab matrix copy <lab-id> <run-id> <task>/<cell>/<repeat> [flags]`
  - An example call is `abcd lab matrix copy lab-260901000000-0123abc mx-260901000000-0a1b feature-a/host-big/1`.
- `abcd lab matrix judge <lab-id> <run-id> <task> [flags]`
  - An example call is `abcd lab matrix judge lab-260901000000-0123abc mx-260901000000-0a1b doc-a --good good.md --bad bad.md --judges 3`.
- `abcd lab matrix report <lab-id> <run-id> [flags]`
  - An example call is `abcd lab matrix report lab-260901000000-0123abc mx-260901000000-0a1b`.
- `abcd lab matrix rerun <lab-id> <run-id|run-folder> [flags]`
  - An example call is `abcd lab matrix rerun lab-260901000000-0123abc mx-260901000000-0a1b`.
- `abcd lab matrix review <lab-id> <run-id> <task> [flags]`
  - Reads `--answers`, the interview answers.
  - An example call is `abcd lab matrix review lab-260901000000-0123abc mx-260901000000-0a1b feature-a`.
- `abcd lab mint <question> [flags]`
  - An example call is `abcd lab mint "does the snapshot keep the checkout's hooks from firing?"`.
- `abcd lab preflight <lab-id> [flags]`
  - An example call is `abcd lab preflight lab-260901000000-0123abc`.
- `abcd lab record <lab-id> <probe> [flags]`
  - An example call is `abcd lab record lab-260901000000-0123abc bare-status`.
- `abcd lab sweep <lab-id> [flags]`
  - An example call is `abcd lab sweep lab-260901000000-0123abc`.
<!-- /generated -->

**User input:** $ARGUMENTS
