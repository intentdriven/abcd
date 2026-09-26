---
name: lab
description: "List this repository's labs with their pins, probe counts and halts: Writes nothing; refuses outside a git checkout."
block: agents
---

# `/abcd:lab`

Run a lab against a pinned snapshot of this repository. The lab's evidence lives
in the machine-scoped lab store (`~/.abcd/lab/<root-sha>/<lab-id>/`), its
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
conditions before anything mutates, and build the work binary once from the
pristine snapshot into `bin/abcd`.

**Preflight** the lab before any mutation:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" lab preflight --json <lab-id>
```

`checks` holds the harness-isolation group (`isolation.home`,
`isolation.snapshot`, `isolation.remotes`, `isolation.hooks`) and the dual-binary
group (`binary.work`, `binary.pinned`, `binary.test`), each with `ok` and
`detail`; the artefact is `state/preflight.md`. On any failure `passed` is false,
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
`go run ./cmd/abcd ahoy install` in a source checkout.

**User input:** $ARGUMENTS
