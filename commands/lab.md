---
name: lab
description: "List this repository's labs with their pins, probe counts and halts: Writes nothing; refuses outside a git checkout."
block: people
---

# `/abcd:lab`

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
For each type, `agreement` gives `pairs`, `compared` (the adapter answered),
`agreed`, `skipped` (the state was over the model's window, or the call
failed), `rate`, and the compared and agreed counts in four probability
`bands`. It ends with the `note` line, naming the lab home and the count. The
drain's judgement before a lane opens is the type `eligibility-residual`. Show
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
declares `minutes_per_copy`, otherwise known after the pilot), `judges`, and the
`digest` naming exactly this plan, with `approve_with`, the command that starts
it. Building it makes no network call and fetches no price. Show the person the
whole plan in plain words: the copies, what it may spend against the cap, the
load, the time and the judges.

**Only after the person has seen that plan and said yes to it**, record the run
with the digest it carried:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" lab matrix --json <lab-id> <declaration.json> --approve <digest>
```

The plan is built again and must carry that digest; an approval naming another
plan is refused with nothing written. Never pass `--approve` on your own
judgement, and never with a digest the person did not see. Starting the run's
copies is not built yet, so a consented run is recorded in the lab and nothing
is launched: `launched` is false and `note` says so; tell the person that
plainly. Refused, exit 2, with nothing written: a metered cell with no spending
cap (set one in the declaration's `spend_cap` or `matrix.spend_cap` in the
person's own machine configuration, never the repository's), a local cell with
no load cap or one at or above the cores, a route this machine's runners refuse,
an OpenRouter or local cell with no pin, and a stale approval.

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

**User input:** $ARGUMENTS
