---
name: disembark
description: Pack a lifeboat from a repository into a destination directory — read-only over the source, behind a destination safety gate, secret-scanned before any write. Point it at any repo (including a dead or archived one) and write the lifeboat elsewhere.
argument-hint: "<source-repo> <dest> | plan <source-repo> | probe <source-repo>"
block: people
---

# `/abcd:disembark` — pack a lifeboat

Mine a repository's record into a portable lifeboat at `<dest>`. The source is
**never written** — a probe and plan read it read-only, and the pack writes only
to the destination. This is the out-of-tree model (adr-35): point it at any repo,
touch nothing, write elsewhere.

## Probe first (coverage readout)

See which brief sections a lifeboat could ground from the source, without writing
anything. The repo argument is optional and defaults to the current directory:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" disembark probe <source-repo> --json
```

This is the coverage experiment's read-only readout: every brief section comes
back marked `grounded`, `partial`, or `blank`, alongside what was searched. It
writes nothing into the source and runs in a small fraction of a full pack's time
(no delegated model work). `coverage.{json,md}` are written only by `pack`.

## Dry run next (recommended)

Show the exact file set a pack would write, without writing anything:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" disembark plan <source-repo> --json
```

Report `file_count`, `total_bytes`, `manifest_sha256`, and any `omissions` (records
too large or unreadable to carry). Then pack for real.

## Scope: what the scan reads

The free-text tree scan — the one that grounds evidence and open-question
markers by reading source files — honours `.gitignore` by default. A file the
user told git to ignore is out of that scan, because a lifeboat cites its
evidence by `path:line` and is meant to be shared, so a scan that read ignored
files could carry a repository's scratch, logs and local notes into the
artefact. Declared record families (ADRs, issues, intents, specs) are a
separate path: `pack` copies them verbatim from their canonical locations
whether or not git ignores them, so a gitignored record still travels — keep a
record you do not want shared out of those locations, not merely in
`.gitignore`.

The salvage case is the other way round, and it is the case `disembark` exists
for: a **dead or archived** repository whose uncommitted residue is often the
most valuable thing left. Reach that with an explicit flag on any of the three:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" disembark probe <source-repo> --include-ignored --json
```

**Offer it; never assume it.** Widening the scan is the user's choice to make,
not a default to infer from a repo looking abandoned. When a probe comes back
thin over a repo that plainly had work in it, say the scan honoured `.gitignore`
and ask whether to widen — do not re-run wide on your own judgement.

The wide scan declares itself: the report carries `included_ignored: true`
(`scope: WIDE` in the text rendering), and the marker scan's `searched` line says
which scope it ran at. A default report says nothing, because the narrow scan is
what every reader already assumes. This flag reports the tree scan's scope only —
it does not track the record-family copy above, which runs the same either way.

## Coverage (aggregate probe reports)

Aggregate one or more saved probe reports into the cross-repo section×repo
coverage table — the experiment's evidence that the packer's section list holds
across a rich-record repo and a git-only one:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" disembark coverage <report.json>... --json
```

Each positional argument is a probe report emitted with `probe --json`.

## Pack

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" disembark pack <source-repo> <dest> --json
```

Summarise the JSON result for the user:

- `dest` — where the lifeboat was written.
- `files_written` / `bytes_written` — the size of the lifeboat.
- `manifest_sha256` — the pinned hash over every file (matches `<dest>/_provenance.json`).
- `voyage_appended` — whether the operator-level voyage ledger recorded the pack
  (`~/.abcd/voyage/<source-root-sha>/disembark/history.jsonl`); `voyage_note`
  explains a skip (e.g. a source with no root-commit SHA).
- `omissions` — any records deliberately left out, declared rather than dropped.

## What the pack refuses

The **destination safety gate** protects real work. A pack refuses unless `<dest>`
is absent, an empty directory, or an existing lifeboat abcd produced (it carries a
parseable `_provenance.json`). It also refuses a symlinked destination, one inside
a `.git/` directory, or one that overlaps the source tree. And it **refuses on a
hard-fail secret** in the planned bytes — a secret is fixed at source, never
redacted into the artefact. Relay the refusal message so the user knows what to fix.

## Graveyard interpretation (layer 3)

A packed lifeboat carries a **graveyard** of what the project tried and left
behind: `graveyard/archaeology.json` (deterministic git evidence — reverts,
unmerged branches, deleted paths, removed dependencies, wholesale rewrites) and
`graveyard/abandoned.json` (what the record itself declared dead — superseded
intents and ADRs, wontfix issues, rejected options). These are evidence only; no
interpretation.

To add interpreted lessons, run the `graveyard-interpreter` agent over those two
files, have it emit a lesson JSON document (each lesson **citing** the finding ids
it rests on), write that document to a file, then:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" disembark graveyard <lifeboat-dir> --lessons-json <path>   # or - for stdin
```

The verb is a **cite-or-be-dropped** gate. A lesson survives only if at least one
of its `evidence` refs resolves to a live finding id from layers 1/2; a lesson
that cites nothing (or only dead refs) is **dropped** — reported in the result,
never fatal. Surviving lessons are written to `graveyard/lessons.json`, sorted by
id. A lesson marked `confidence: "low"` is routed to
`graveyard/low-confidence/<id>.json` instead, kept apart from the confident set.
Re-ingesting **replaces** the prior interpretation: each run rewrites layer 3
from the current payload's survivors, so a lesson promoted low→high or dropped
from a later payload leaves nothing stale behind.

Report the result to the user: `written` (into `lessons.json`), `low_confidence`
(routed aside), and `dropped` (with the reason for each). The verb **exits 0 even
when every lesson was dropped** — an honest "nothing cited" is a valid outcome.
It exits non-zero only on a structural fault: the directory is not an abcd
lifeboat, its graveyard files are unreadable, or the lesson payload is
unreadable, oversize, or malformed. The lesson files are a later, mutable
interpretation and are **not** part of the lifeboat's `manifest_sha256`.

## Principles (distilled from the record)

A packed lifeboat can carry **principles** — the load-bearing decisions the project
settled, each citing the record it rests on. Run the `principle-distiller` agent
(`agents/principle-distiller.md`) over the packed `docs/adrs/` and
`activity/issues/`, have it emit a principle JSON document (each principle
**citing** the record ids — `adr-N`, `itd-N`, `iss-N` — or the lifeboat paths it
distils from), write that document to a file, then:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" disembark principles <lifeboat-dir> --principles-json <path>   # or - for stdin
```

**Without the flag** the verb runs deterministic mode: it writes an evidence-only
`principles.json` composed straight from the packed ADRs' own stated decisions —
no agent, no interpretation, byte-identical across re-runs.

With the flag it is a **cite-or-be-dropped** gate. A principle survives only if at
least one of its `evidence` refs resolves to a live record/finding id or a packed
lifeboat path; one that cites nothing resolvable is **dropped** — reported in the
result, never fatal. Surviving principles are written to `principles.json` (sorted
by id) and rendered to `principles.md`; a delegated ingest **fully replaces** the
prior file. Report `written` and each `dropped` (with its reason). The verb
**exits 0 even when every principle was dropped** — the file is written with an
empty list, an honest "nothing distilled". It exits non-zero only on a structural
fault (not an abcd lifeboat, or a payload that is unreadable, oversize, or
malformed). `principles.json`/`.md` are a later, mutable synthesis layer and are
**not** part of the lifeboat's `manifest_sha256`.

## Press release (the embark interview contract)

The lifeboat's **press release** is the one-paragraph "what this project is"
statement a future embark interview reads back. Run the `press-release-composer`
agent (`agents/press-release-composer.md`) over the packed `brief/`,
`rescue/spine.md`, and `principles.json`, have it emit a press-release JSON
document whose `evidence` cites those inputs, write it to a file, then:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" disembark press-release <lifeboat-dir> --press-release-json <path>   # or - for stdin
```

**Without the flag** the verb composes deterministically from the packed brief
press-release section (falling back to the spine, then to a grounded-nothing
placeholder) and writes `press-release.json` + `.md`. Always written,
byte-identical across re-runs.

A press release is a **single document**, not a list, so the citation rule is
whole-document: its `evidence` must carry at least one ref resolving to `brief/**`,
`rescue/spine.md`, or `principles.json`. A delegated document that cites **nothing
resolvable is refused** — the verb exits non-zero and leaves the previously derived
press release untouched (there is no per-entry granularity to drop). Report the
result: `mode` and `evidence_refs`. It also exits non-zero on a structural fault
(not a lifeboat, or an unreadable/oversize/malformed payload). The press-release
files are a mutable synthesis layer and are **not** part of `manifest_sha256`.

## Review (content fidelity + verdict)

The **review** assesses a packed lifeboat against the source repo it was mined from
and issues a registered verdict — `SHIP`, `NEEDS_WORK`, or `MAJOR_RETHINK` — with
findings that each cite a lifeboat file. Run the `lifeboat-reviewer` agent
(`agents/lifeboat-reviewer.md`) over the whole packed lifeboat versus the source,
have it emit a verdict JSON document (verdict + findings, each **citing** a packed
lifeboat path), write it to a file, then:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" disembark review <lifeboat-dir> <source-repo> --review-json <path>   # or - for stdin
```

The `<source-repo>` argument is **required**; its content is never read — the
binary gates it as a real directory only, so the review stays deterministic and
safe even when the source is gone. The binary always **stamps the attestation
fields itself** (source name, manifest hash, manifest-verified flag, coverage
summary) — they are never taken from the agent's payload.

**Without the flag** the verb computes a deterministic verdict from manifest
verification plus the packed coverage summary: a lifeboat whose bytes no longer
match its seal is `MAJOR_RETHINK`; one that cannot attest coverage, or whose blank
sections outnumber its grounded ones, is `NEEDS_WORK`; otherwise `SHIP`. A manifest
mismatch is a **verdict input, not a failure** — the review is still written and the
verb exits 0.

With the flag, the model's `verdict` is **membership-checked** (an out-of-enum
verdict is refused, exit non-zero) and its findings are **cite-or-be-dropped**
against the packed path set. The review lands at
`review/review-<manifest12>.json` + `.md`, named from the manifest hash (no
timestamp), so a deterministic run and a later delegated run write the **same**
file — a clean replacement, no stale twin. Report the `verdict`, `written`
findings, and each `dropped` (with its reason); the verb exits 0 even when every
finding was dropped. It exits non-zero only on a structural fault (not a lifeboat,
a `<source-repo>` that is not a real directory, or an unreadable/oversize/malformed
payload). The audit files are a mutable synthesis layer and are **not** part of
`manifest_sha256`.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

**User input:** $ARGUMENTS
