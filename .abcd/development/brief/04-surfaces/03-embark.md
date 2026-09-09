# `/abcd:embark` — Unpack a Lifeboat

> **Delivery state (shipped surface).** The binary ships exactly two `embark`
> sub-verbs — **`from <lifeboat> [target]`** and **`probe <lifeboat> [target]`** —
> and `from` takes **no flags** (only the global `--json`). The richer surface
> this chapter designs — the **`scan`** discovery sub-verb (and its `--deep`),
> and the `from` flag-shaped modifiers **`--force`**, **`--archive`**, and
> **`--refresh-audit`** — is a **design target, not yet shipped**; every
> present-tense mention of them below describes the intended design, not current
> binary behaviour. This chapter is the design record for the full surface; the
> shipped subset is `from` + `probe`.

> **⚠ Partly superseded by [adr-35](../../decisions/adrs/0035-lifeboat-as-coverage-experiment.md).** The reconciled prose lands with the unpacker (the `embark` verb and `commands/embark.md` now ship — with adr-35's out-of-tree model, not this chapter's in-tree prose). Four changes:
>
> | This chapter says | adr-35 decides |
> |---|---|
> | `voyage/` in-tree at `.abcd/development/voyage/` (per adr-4) | **Operator level**: `~/.abcd/voyage/<source-root-sha>/`, keyed on the root-commit SHA like the history store — never committed. This dissolves the `privacy-hygiene` collision: voyage records absolute source paths, and abcd's own audit rule flags those in committed files. |
> | `from home` reads the current repo's `.abcd/lifeboat/` | There is **no in-tree lifeboat home** and no `home`. Disembark writes out-of-tree to an operator-chosen `<dest>`; embark reads from wherever that was. |
> | Writes land via ordinary file writes | Writes go through **`os.Root` containment plus independent path validation** — two layers, so a bug in one is not a CVE. A leaf-only `O_NOFOLLOW` is insufficient: a symlinked *intermediate* directory still escapes. |
> | A refusal path *writes* a conflict report | **Core must not write on a refusal path** (transport-agnostic-core violation). Core returns the conflicts; the surface renders them, as one bulk prompt rather than a per-file barrage. |
>
> One more, and it is the load-bearing one: **lifeboat text is never injected verbatim into `CLAUDE.md`.** The *current* marker block is re-injected instead. That line is the difference between a data leak and a persistent instruction implant — lifeboat content otherwise lands in a file the agent obeys.

> **Phase ownership** ([adr-33](../../decisions/adrs/0033-launch-phase-ownership-tiered.md)): the lifeboat round-trip — disembark packing and embark unpacking — ships in [Phase 6](../../roadmap/phases/phase-6-lifeboat.md).

> **Recovery humility.** Embark unpacks the lifeboat into a working repo. The lifeboat is the highest-fidelity floor the originating session could leave behind; it is not the activity that produced it. **When something here doesn't make sense, hunt the originating session before trusting the lifeboat blindly** — ask the prior author, surface the chat where the decision happened, look at the rejected alternatives. The lifeboat is a starting point, not an oracle. See [`01-product/03-mental-model.md § The Naurian gap`](../01-product/03-mental-model.md#the-naurian-gap--modification-axis) for the framing.

## Sub-verbs

> _Machine-checked (`surface_coverage`, spc-27): each row records the verb's
> adr-40 bucket (`lint` / `review` / `audit` / `gate`, or `—` for a
> non-assessment verb) and its existence (`shipped` / `staged`), verified
> against the committed command-tree snapshot in both directions._

| Verb | Bucket | Status |
|---|---|---|
| `from` | — | shipped |
| `probe` | — | shipped |


Bare `/abcd:embark` prints dispatcher help only — never mutates state. The two **shipped** sub-verbs:

- **`/abcd:embark from <path> [target]`** — unpack the lifeboat at `<path>` into `[target]`, which defaults to the working directory when omitted. `<path>` is required, and it is always an explicit path to a destination a disembark wrote — **there is no `home` shorthand** (adr-35: there is no in-tree lifeboat home to expand it to). The round-trip / self-test case is `disembark pack <repo> <dest>` followed by `embark from <dest>`. *(Design target, not yet shipped: the flag-shaped modifiers `--force` — override conflict refusal, `--archive` — copy input lifeboat verbatim to `~/.abcd/voyage/<source-root-sha>/embark/from/<timestamp>/` before unpacking, and `--refresh-audit` — re-run oracle product audit instead of trusting cached. The shipped `from` takes no flags.)*
- **`/abcd:embark probe <path> [target]`** — inspect what the lifeboat at `<path>` would write into `[target]` (defaulting to the working directory) without unpacking: show what would land where, verify the lifeboat against its `manifest_sha256`, refuse a `schema_version` this build does not understand, write nothing. No product audit runs here: the oracle re-audit is the design-target `--refresh-audit` modifier above, and `audit/**` is read as a report-only path.

Design-target sub-verb (not yet shipped):

- **`/abcd:embark scan`** — discovery sub-verb: list **lifeboat destinations** — directories carrying a parseable `_provenance.json`, the same marker the destination safety gate keys on (adr-35) — ranked by mtime, presented as candidates via transparent prompt. **No unpacking.** Useful before `embark from <path>` when the user isn't sure where lifeboats live. Flag-shaped modifier: `--deep` (a wider walk for power users).

> **Open question (adr-35):** where `scan` searches. Walking `../` made sense when a lifeboat lived inside its producing repo, so siblings-of-cwd *were* the candidate set. Destinations are now operator-chosen and need not sit beside the repo being embarked into. Either the sibling walk is kept as a cheap heuristic, or scan is given explicit roots (an argument, a configured search path, or the voyage records under `~/.abcd/voyage/`). adr-35 does not settle this; it must be decided before `scan` is specified — and the same note is carried in [`02-constraints/01-platform.md § Embark sources`](../02-constraints/01-platform.md#embark-sources). The depth semantics of `--deep` fall out of whatever that decides.
- **Later phase: `/abcd:embark from-spec-kit <path>`** — ingest a GitHub Spec Kit project directory as starter draft intents (per itd-23); not yet shipped.

## 1. Source lookup

Per [`02-constraints/01-platform.md § Lifeboat path`](../02-constraints/01-platform.md#lifeboat-path), lifeboats are *output*, and they land **out-of-tree at an operator-chosen destination** — `disembark pack <source-repo> <dest>` never writes to the source repo. There is no in-tree lifeboat home and no repo-local registry of inbound lifeboats. Embark therefore always reads from an external source: the destination some disembark wrote to.

Path resolution under `from <path>`:

1. `<path>` → validate, use. There is no `home` shorthand (adr-35). The round-trip / self-test case is not special-cased either: it is `disembark pack <repo> <dest>` followed by `embark from <dest>`, the same explicit path as any other source.
2. To *find* candidate lifeboats before running `from <path>`, use `embark scan` (or `embark scan --deep`) — that's a separate sub-verb, not a flag on `from`.

**Provenance and `--archive`** *(design target, not yet shipped)*: `embark from <path>` records `source_path` and `source_manifest_sha256` in `~/.abcd/voyage/<source-root-sha>/embark/provenance.json` (see [§ 7](#7-voyage-layout--embarkdisembark-provenance-and-history)). Opt-in `embark from <path> --archive` additionally copies the input lifeboat verbatim into `~/.abcd/voyage/<source-root-sha>/embark/from/<timestamp>/` for the case where the source repo will disappear. Off by default; `source_path` + hash is enough when the source repo persists.

No global `~/.abcd/archive/`.

## 2. Conflict-based refusal

There is no emptiness gate. A conflict is per-file: a target that merely carries unrelated files is not a conflict, and a planned file whose bytes already match is an idempotent skip. Embark refuses only when a planned target path already holds differing bytes, is a non-regular target, is a duplicate target, or sits under a non-directory parent. On **any** such conflict **core writes nothing** — it *returns* the conflict set it found, and the surface renders it (adr-35: a refusal that writes a file is a transport-agnostic-core violation). *(Design target, not yet shipped: `embark from <path> --force` to proceed to conflict resolution ([§ 4](#4-conflict-ux)); the shipped `from` takes no flags.)*

## 3. Scaffold steps

Embark is a deterministic Go run: it reads the lifeboat, plans, refuses on any conflict, then writes the record families plus the `CLAUDE.md` marker. No interactive scaffolder, no Python, and no LLM agent sit in the write path.

0. **Read lifeboat:** the record families — ADRs (`docs/adrs/`), issues (`activity/issues/`), intents (`rescue/intents/`), specs (`rescue/specs/`) — plus the report-only files `_provenance.json`, `press-release.{json,md}`, `principles.{json,md}`, `coverage.{json,md}`, `brief/**`, `rescue/spine.md`, `graveyard/**`, and the post-pack `review/**` verdict artefact (`audit/**` on a pack made before spc-30). The lifeboat is untrusted input: embark verifies its `manifest_sha256` against the on-disk tree — every hashed file, i.e. the tree minus the post-pack synthesis layer the manifest deliberately excludes (`_provenance.json`, `press-release.{json,md}`, `principles.{json,md}`, `review/**`, `audit/**`, `graveyard/lessons.json`, `graveyard/low-confidence/**`) — and refuses a symlink or oversize file anywhere inside. A tampered hashed record or an added stray file is refused; the excluded report-only files sit outside the manifest seal and carry their own per-entry integrity (cite-or-be-dropped, registered-verdict gate), not the hash.
1. **Plan.** Map each record file to its target path and classify it `create`, `unchanged` (byte-identical), or a conflict. On **any** conflict embark writes nothing and refuses ([§ 4](#4-conflict-ux)).
2. **Write the record families verbatim** to their canonical target locations — ADRs to `.abcd/development/decisions/adrs/`, issues to `.abcd/work/issues/`, intents to `.abcd/development/intents/`, specs to `.abcd/development/specs/` — through two-layer containment (an `os.Root` boundary plus independent lexical path validation), skipping `unchanged` files. Bucketed families keep their source bucket (issues by state; intents into `drafts`/`planned`/`shipped`/`disciplines`/`superseded`; specs into `open`/`closed`). Terminology, docs, and `.abcd/memory/` are **not** embark families — they do not travel.
3. **Re-inject the current abcd marker block** into the target `CLAUDE.md` between BEGIN/END markers (idempotent) — never AGENTS.md, and never a verbatim copy of lifeboat prose. The block is the modular-rules-loader block (per itd-3); principles surface through the rules loader's domain rules on demand by prompt-keyword recall.
4. **Report** the outcome to the surface, blanks first: any pass the lifeboat declares exempt ([§ 5](#5-the-coverage-handoff)), the coverage blanks a human must answer, then `embarked <source> into <target>`, the `written`/`unchanged` counts with the per-family breakdown, and the marker action. The report-only files that informed the run are tallied by `embark probe` alone: the `ignored` slice rides on the result and is reachable through `--json`, but the rendered `from` report omits it, because a human reading the outcome of a write wants what landed, and the probe is where a human asks what a lifeboat holds.

*(Design target, not yet shipped: write voyage provenance to `~/.abcd/voyage/<source-root-sha>/embark/provenance.json` — see [§ 7](#7-voyage-layout--embarkdisembark-provenance-and-history).)*

## 4. Conflict UX

When the target already has files that the lifeboat would write, **core returns the conflict set and the surface renders it as a single bulk report** — never a per-file barrage, and never a file written by core (adr-35). The shipped `from` **writes nothing** on any conflict and exits non-zero; the surface relays the bulk report so the user resolves the conflicts and re-runs:

```
2 conflict(s) (nothing was written):
  • .abcd/development/specs/open/spc-12.md  (exists-differs)
  • .abcd/work/issues/open/iss-7.md  (exists-differs)

nothing was written — resolve the conflicts and re-run
```

The report is a per-file list — one line per conflicting target path with its
conflict kind (`exists-differs`) — not a per-family count roll-up.

The conflict list is a value core hands back; if the operator wants it on disk, the surface writes it — core does not.

*(Design target, not yet shipped: `embark from <path> --force` turns that bulk report into a single resolution prompt — keep target (skip everything in lifeboat that conflicts) / replace target (lifeboat wins everywhere) / merge where possible, prompt otherwise / abort (surface prints the conflict list; nothing is written) — and applies the chosen resolution uniformly. Single decision, transparent (shows scope before asking). The shipped `from` takes no flags and offers no resolution choice.)*

## 5. The coverage handoff

Both `probe` and `from` open on what a human still owes the record, before any
write summary. Two things print there, in this order.

**A declared pass exemption.** `_provenance.json` carries an optional
`pass_b_exemption` object, whose one field is a `reason`. Pass B is the pass
that mines chat transcripts for the rationale nobody wrote down; no source tier
this build packs reads a transcript store, so `disembark` writes the
declaration into every pack it produces, and embark carries it through to
`CoverageHandoff.PassBExemption`. It renders as the first line of both reports:

```
pass B (transcripts): declared exempt — no transcript source was read for this package, so the rationale Pass B would have mined is absent rather than omitted
```

The declaration outlives the blanks below it, and that is the point of it being
a declaration rather than a silent gap: a brief section Pass B would have
grounded reads as a pass that never ran, not as work a human has been left. The
field is an omitted pointer rather than a bool, so a package that predates it
marshals exactly as it always did, and the branch that stops the declaration is
exercised before any transcript adapter exists: an exemption that could not
stop would become a false claim in a durable artefact the day Pass B ships.

**The coverage blanks.** Past the exemption, an absent `coverage.json` prints
nothing, a degraded one prints a one-line note, and a present one prints each
unanswered brief section with the question that grounds it, human-owned
sections marked as yours to write. A present coverage with no blanks prints
nothing: there is nothing to answer.

## 6. Acceptance

> **Scope:** criteria naming the design-target surface (the `scan` sub-verb and
> its `--deep`, and the `from` flags `--force` / `--archive` / `--refresh-audit`)
> are gated on that surface shipping — they describe the intended behaviour, not
> the current binary (which ships `from` + `probe`, no flags). The shipped-surface
> criteria (`from`, `probe`, bare invocation, the conflict refusal) hold today.

- **Given** any abcd-aware terminal, **when** the user runs bare `/abcd:embark`, **then** the dispatcher prints help listing the shipped sub-verbs (`from`, `probe`) and the global `--json` flag — bare invocation never mutates state.
- **Given** a lifeboat at `<path>` and a conflict-free target, **when** `/abcd:embark from <path> [target]` runs (the target defaulting to the working directory when omitted), **then** the four record families (ADRs, issues, intents, specs) land at their canonical locations under the target, the current abcd marker block is re-injected into the target `CLAUDE.md`, and everything else in the lifeboat informs the report but is never written.
- **Given** a repo disembarked to `<dest>`, **when** `embark from <dest>` runs in an empty target, **then** the round-trip completes with no shorthand and no special case — `<dest>` is an ordinary explicit path. (There is no `home`, and no sub-verb resolves one; matches disembark's parallel rule for sub-verb-agnostic resolution.)
- **Given** a target holding a file that conflicts with a planned write, **when** `/abcd:embark from <path>` runs, **then** the command refuses, core returns the conflict list **without writing any file**, and the surface renders it as one bulk report — a target that merely holds unrelated files is not a conflict.
- **Given** a non-empty target with conflicts and `--force`, **when** `embark from <path> --force` runs, **then** the bulk conflict prompt fires once with a summary of all conflicts, and the chosen resolution is applied uniformly.
- **Given** the user runs `/abcd:embark scan`, **when** the command completes, **then** the directories found carrying a parseable `_provenance.json` are listed ranked by mtime with their detected source repo, no unpacking occurs, and the user is shown candidates ready to pass to `embark from <path>`. *(What `scan` searches is settled; **where** it searches is the open question above, and this criterion cannot be made checkable until that is decided.)*
- **Given** the user runs `/abcd:embark scan --deep`, **when** the command completes, **then** the search widens — the exact widening depends on the same open question.
- **Given** the user runs `/abcd:embark probe <path> [target]` (the target defaulting to the working directory when omitted), **when** the command completes, **then** the lifeboat at `<path>` is inspected against the target (file tree, schema validation, would-be writes), no target mutation occurs, and the user sees a report ready to inform the decision to run `embark from <path> [target]`.
- *(Design target, not yet shipped.)* **Given** a lifeboat containing `.abcd/rp/workspace.json` and RP installed on the embarker, **when** `embark from <path>` runs, **then** the user is asked whether to register the workspace with RP and the choice is applied.
- *(Design target, not yet shipped.)* **Given** a lifeboat containing `.abcd/rp/workspace.json` and RP *not* installed, **when** `embark from <path>` runs, **then** the command warns gracefully and continues without failing.
- **Given** the user passes `--refresh-audit`, **when** `embark from <path> --refresh-audit` runs, **then** the oracle product audit re-runs against the current lifeboat content and the drift vs the disembark-time audit is reported.
- *(Design target, not yet shipped.)* **Given** an `embark from <path>` run completes, **then** `~/.abcd/voyage/<source-root-sha>/embark/provenance.json` exists with `source_path`, `source_manifest_sha256`, `timestamp`, and `files_written` populated (per [§ 7](#7-voyage-layout--embarkdisembark-provenance-and-history)); `embark/from/<timestamp>/` is absent unless `--archive` was passed.
- **Given** an `embark from <path> --archive` run completes, **then** the input lifeboat is copied verbatim to `~/.abcd/voyage/<source-root-sha>/embark/from/<timestamp>/` and the path is referenced from `provenance.json`.

## 7. Voyage layout — embark/disembark provenance and history

Lifeboat *operations* (embark, disembark) write provenance and history to **`~/.abcd/voyage/<source-root-sha>/`** — the operator level, keyed on the root-commit SHA exactly as the history store is, and therefore never committed (adr-35, superseding adr-4's in-tree `.abcd/development/voyage/`). The lifeboat itself is written **out-of-tree** to the operator-chosen `<dest>` ([`02-disembark.md § 5`](02-disembark.md#5-output-shape)) and holds only the latest snapshot; it does not accumulate.

```
~/.abcd/voyage/<source-root-sha>/            ← operator level, keyed like the history store; never committed
├── embark/                                  ← design target, not yet shipped
│   ├── provenance.json                      ← source path, manifest hash, timestamp, files written
│   └── from/<timestamp>/                    ← --archive: verbatim copy of input lifeboat (opt-in)
└── disembark/
    └── history.jsonl                        ← append-only manifest log of every disembark
```

Keying on the source repo's root-commit SHA is what lets voyage survive a rename, a remote move, or the source repo being deleted entirely — and it is why voyage may hold absolute source paths without ever putting them in a committed file.

**`embark/provenance.json`** *(design target, not yet shipped)* records, for the embark that bootstrapped this repo:

- `source_path` (the `<path>` argument passed to `embark from <path>`)
- `source_manifest_sha256` (hash of input lifeboat's `_provenance.json` + file tree)
- `timestamp`
- `files_written` (target paths created for the record families — ADRs at `.abcd/development/decisions/adrs/`, issues at `.abcd/work/issues/`, intents at `.abcd/development/intents/`, specs at `.abcd/development/specs/`)
- `audit_drift` (only if `--refresh-audit`: drift vs disembark-time `audit/press-release-oracle-*`)

**`embark/from/<timestamp>/`** — opt-in via `embark --archive`. Verbatim copy of the input lifeboat at the moment of embark, for the case where the source repo will disappear. Off by default; the `source_path` + hash in `provenance.json` is sufficient when the source repo persists.

**`disembark/history.jsonl`** — append-only, one JSON object per disembark run:

```json
{
  "schema_version": 2,
  "event": "disembark",
  "at": "2026-05-04T14:30:00Z",
  "manifest_sha256": "abc123...",
  "source_name": "abcd-cli",
  "source_root_sha": "def456...",
  "dest": "../abcd-lifeboat",
  "files": 214,
  "bytes": 1048576
}
```

> **The entry is deliberately minimal** (adr-35): it carries the manifest identity, the source name and root SHA, the destination, and the file/byte counts — not a verdict, a label, or a `shared_with` recipient list. An empty field is a lie in a schema; add any of these when something actually populates it.

Manifests are small (file list + hashes, not contents); the log answers "what did this repo's lifeboat look like at point T?" without keeping stale snapshots around. Acceptance criteria for voyage writes live in [`02-disembark.md § 7`](02-disembark.md#7-acceptance) (disembark-side history.jsonl) and [§ 6](#6-acceptance) above (embark-side provenance.json + optional --archive copy).
