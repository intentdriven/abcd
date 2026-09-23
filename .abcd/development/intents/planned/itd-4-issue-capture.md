---
id: itd-4
slug: issue-capture
spec_id: spc-6
kind: standalone
suggested_kind: null
reclassification_history: []
severity: major
---

# Nothing You Notice Gets Lost

## Press Release

> **abcd turns the steady drip of nitpicks, review findings, and "huh, that's odd" moments into a queryable ledger.** A new `/abcd:capture` command writes structured `iss-N` entries in seconds; the entries live in a committed `.abcd/work/issues/` ledger with stable IDs (`iss-N`) and folder-as-status (`open/`, `resolved/`, `wontfix/`). The `Mandatory Issue Recording` rule no longer depends on the agent remembering — there's a fast deliberate path with a structured destination, and the ledger is ready for the cross-corpus synthesist (`/abcd:dredge`, see itd-25) to mine once enough data has accumulated.
>
> "I do thorough plan and implementation reviews and they always surface nitpicks the agent files away in the transport output but not on disk," said Maya, autonomous-development practitioner. "I'd write 'remember to log this' a dozen times a session and still lose half of them. With abcd, I just say 'capture: T7 cache_ttl_days dead-config alternative' and it's structured, IDed, and queryable. The synthesis layer (itd-25) depends on the ledger having enough volume to be worth synthesising — but the capture surface is there from day one."

## Why This Matters

The `~/ABCDevelopment/.claude/CLAUDE.md` `Mandatory Issue Recording` rule is correct in spirit but unreliable in practice. Reviews and manual testing surface nitpicks that vanish unless the persona explicitly tells the agent to log them. When they *are* logged, `.abcd/.work.local/issues.md` is gitignored, per-repo, free-form Markdown — there's no way to ask "what categories of finding recur across my repos and over time?".

Two distinct problems compound: **capture is unreliable** (during-work) and **synthesis doesn't exist** (over-time, cross-repo). They have very different value-timing: capture earns its keep on day one; synthesis only earns its keep once a meaningful ledger has accumulated. This intent ships **capture only**. The cross-corpus synthesist that uses the ledger as input is a separate intent ([itd-25](../drafts/itd-25-dredge-cross-corpus-synthesist.md)), deferred until the ledger has sufficient volume.

This split closes a separate loop on `/abcd:capture` as a command name. The maritime-metaphor review for `/abcd:intent` established that "capture" is too neutral for product-framing intent work — but it's *exactly right* for issues, where the verb genuinely shouldn't pre-commit to whether a finding is a bug, nitpick, or systemic pattern. The synthesist (itd-25) decides that later. So abcd lands `/abcd:capture` not as a rename of `/abcd:intent` but as its sibling. Two meta-development surfaces with distinct jobs: `intent` frames product, `capture` ingests signal. `capture` and `intent` are metaphor-exempt; `dredge` (itd-25) rejoins the maritime convention as the cross-corpus counterpart to `lifeboat`.

**Brief-revision dependency:** Promoting this intent invalidates several "5 commands" claims in the brief body. Brief revisions land in the accompanying brief rewrite, not at intent-plan time.

## What's In Scope

- **`/abcd:capture` command (6th plugin command)** — single-entry ingest, with subverbs:
  - `capture <text>` — fast path; appends a structured `iss-N` entry to the ledger with auto-assigned `iss-N`, timestamp, and source provenance (which session / command / file the persona was in)
  - `capture list [--open|--resolved|--wontfix|--all]` — query the ledger
  - `capture promote <iss-N>` — promote an `iss-N` entry to an intent draft (calls `/abcd:intent new` with the entry body + bidirectional link)
  - `capture resolve <iss-N>` — move to `resolved/` with optional resolution note
  - `capture wontfix <iss-N> <reason>` — move to `wontfix/` with explicit decision
- **`iss-N` ledger structure** at `.abcd/work/issues/`:
  - `open/iss-N-<slug>.md` — captured, not yet acted on
  - `resolved/iss-N-<slug>.md` — fixed (with resolution notes)
  - `wontfix/iss-N-<slug>.md` — explicit non-action decision
  - Frontmatter schema: `id`, `slug`, `severity`, `category`, `source` (review / manual-test / drift / nitpick / observation), `found_during`, `found_at` (path), `related_intents` (list of `itd-N`), `related_epics` (list of `spc-N`), `created`, `updated`
  - Folder-as-status (mirrors intent and spec-roadmap conventions)
  - Stable `iss-N` IDs (unpadded; mirrors `itd-N` convention; lexical-vs-numeric sort handled at tool layer)
- **Brief § 5 update** (lands in the accompanying brief rewrite): exemption note for `/abcd:intent` + `/abcd:capture`; reserved-meta-command table covering `/abcd:dredge` (see itd-25), `/abcd:audit` (reserved, see itd-16 hash-chain-merkle-audit), `/abcd:reflect` (reserved, see itd-24).
- **`.abcd/.work.local/issues.md` migration path**: `dev-sync` promotes existing `.abcd/.work.local/issues.md` entries to the structured ledger on first run after install (or first `/abcd:ahoy` upgrade). Idempotent. Old `.abcd/.work.local/issues.md` becomes a staging buffer (still works for ad-hoc scribbles; promoted on next `dev-sync` or `/abcd:capture promote`).
- **`intent-fidelity-reviewer` extension**: when an intent ships that was promoted from an `iss-N` entry, the reviewer cross-references whether the related `iss-N` entry actually moved to `resolved/`. Mismatch = drift finding.

## What's Out of Scope

- **`/abcd:dredge` cross-corpus synthesist** — a separate intent ([itd-25](../drafts/itd-25-dredge-cross-corpus-synthesist.md)). Capture's value is immediate (every captured `iss-N` entry is useful from day one); dredge's value depends on having an accumulated ledger to synthesise. Shipping dredge without a meaningful ledger produces a synthesist with nothing to synthesise.
- **`issue-synthesist` agent** — belongs to itd-25 (`/abcd:dredge`).
- **Auto-capture hooks** (the "B" tier of the design exploration). Stop-hook / PostToolUse heuristic extraction of `iss-N`-shaped statements from session output. Brittle on output-style drift; risks false-positive fatigue. Defer until the ledger has accumulated real usage data showing whether the manual path's volume is sufficient.
- **Cross-repo `iss-N` copying.** Entries stay in the repo where they were captured. No "promote to a global ledger" behaviour.
- **Severity-based prioritisation UI.** This intent surfaces severity as frontmatter; no dashboard, no SLA, no triage workflow.
- **Renaming `/abcd:intent` to `/abcd:capture`.** Explicitly considered and rejected.
- **Real-time `iss-N` tracking integration** (Linear / GitHub Issues sync). The existing `issue-scout` agent already covers GitHub upstream annotation; sync in the other direction is a separate intent.

## Scope Conditions

None stated.

## Acceptance Criteria

> _BDD format, per `itd-1-acceptance-gates`. These gates are checked by `intent-fidelity-reviewer` when this intent moves to `shipped/`._

- **Given** an abcd-installed repo, **when** the persona runs `/abcd:capture "review nitpick: T7 cache_ttl_days dead-config alternative"`, **then** a new file `.abcd/work/issues/open/iss-N-<slug>.md` exists with frontmatter populated (id, severity, category, source, found_during) and the captured text in the body.
- **Given** an existing `iss-N` entry at `.abcd/work/issues/open/iss-3-foo.md`, **when** the persona runs `/abcd:capture resolve iss-3 "fixed in spc-7 task 4"`, **then** the file moves to `.abcd/work/issues/resolved/iss-3-foo.md` with the resolution note appended to the body.
- **Given** an existing `iss-N` entry, **when** the persona runs `/abcd:capture promote iss-N`, **then** `/abcd:intent new` is invoked with the entry's content as the seed; the resulting intent's frontmatter has `related_issues: [iss-N]`; the `iss-N` entry's frontmatter has `related_intents: [itd-M]` (the new intent's ID). Drift detection enforced by spc-23 (intent-fidelity-reviewer `--issue-drift`).
- **Given** a fresh `/abcd:ahoy` upgrade with an existing `.abcd/.work.local/issues.md`, **when** `dev-sync` runs, **then** every entry in `.abcd/.work.local/issues.md` is promoted to a corresponding `.abcd/work/issues/open/iss-N-<slug>.md` with provenance noting "migrated from .abcd/.work.local/issues.md".
- **Given** the persona runs `/abcd:capture list --open`, **when** there are 5 open `iss-N` entries, **then** the output lists all 5 with id, slug, severity, and one-line summary.

## Open Questions

- **Manual-only capture vs hook-assisted.** This intent defers auto-capture (B) to a future intent. Should a minimal hook ship (e.g., a Stop-hook that simply prompts "Any captures from this session?" rather than auto-extracting), or wait for ledger-volume data first?
- **`iss-N` ID space — per repo, or global across the corpus?** Per-repo is simpler and matches `itd-N` (which is per-plugin-repo). Global would require a registry. Default: per-repo.
- **Promotion from `iss-N` → intent: 1:1 only.** N:1 (multiple `iss-N` entries fold into a single intent) pairs naturally with itd-25 (dredge) and is deferred to that intent. This intent supports 1:1 only.
- **Migration semantics for existing `.abcd/.work.local/issues.md`.** The current file has rich free-form structure (categories in `[brackets]`, "Found while", "Location", "Details", "Suggested fix"). The structured ledger should preserve all of this. Do we auto-migrate (risk: lossy) or interactive-migrate (cost: tedious for ~20 existing entries)?

## Implementing specs

itd-4 was implemented across multiple specs of the superseded pre-Go record
system; those ids are preserved below as history (they do not exist in the
native spec store). The frontmatter `spec_id` records the **native** spec,
**spc-6**, the record catch-up that verifies the shipped engine against the
Acceptance Criteria and carries the open AC3 (promote) gap. Historical index:

- **spc-20** (primary) — `iss-N`-ledger primitives (`iss-N` allocator, schema, capture/resolve/wontfix/update_field workflow, structure under `.abcd/work/issues/`).
- **spc-21** — `/abcd:capture` command surface (flow-text ingest into the ledger).
- **spc-22** — `.abcd/.work.local/issues.md` migration to the structured ledger (`dev-sync work` orchestrator, regex-extracted intent linkage on migrated issues).
- **spc-23** — `intent-fidelity-reviewer --issue-drift` mode (bidirectional cross-reference walk; reader half of the bidirectional contract).

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._
