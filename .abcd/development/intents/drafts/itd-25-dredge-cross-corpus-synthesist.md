---
id: itd-25
slug: dredge-cross-corpus-synthesist
spec_id: null
kind: standalone
suggested_kind: null
reclassification_history: []
blocked_by: [itd-4]
builds_on: [itd-13, itd-36]
severity: minor
---

# Patterns Surface Themselves

## Press Release

> **abcd ships `/abcd:dredge` — the cross-corpus synthesist that turns the issue ledger into systemic insight.** Run it after months of captures have accumulated, and abcd's `issue-synthesist` agent clusters open issues across every repo in `.abcd/corpus.json`, surfacing recurring themes as candidate intent drafts. Three months of captured nitpicks and review findings becomes "nine of your open issues across four repos all touch the same dev-sync race conditions — promote as a single intent?". The thing you were doing in your head is now something the system does for you.
>
> "abcd gave me a structured ledger but I had to scan it manually to spot patterns," said Maya, autonomous-development practitioner. "Dredge looks at every repo's open ledger together, finds the clusters I'd otherwise miss, and offers each as a candidate intent. The first dredge after six months of captures showed me three systemic patterns I'd been working around individually instead of fixing structurally."

## Why This Matters

[itd-4](../planned/itd-4-issue-capture.md) ships `/abcd:capture` and the structured `.abcd/work/issues/` ledger. That's the *capture* half of the original combined intent. The synthesis half — `/abcd:dredge` plus the `issue-synthesist` agent — was deliberately split out because **its value is empty until the ledger has accumulated meaningful data**. Shipping a synthesist with no ledger to synthesise produces a tool that everyone tries once and nobody returns to.

Once the ledger has months of usage across the corpus repos, the synthesist has something to chew on. The pattern echoes the broader abcd philosophy: ship the capture surface that earns its keep on day one; defer the synthesis surface until there's enough data for synthesis to be more than empty ceremony.

The maritime metaphor lands cleanly here: `/abcd:dredge` is the cross-corpus counterpart to `/abcd:disembark`'s lifeboat (per-project rescue ↔ cross-corpus latent-value rescue). Three meta-development surfaces total once both are shipped: `intent` frames product, `capture` ingests signal, `dredge` surfaces patterns.

**Day-one local subset.** The cross-corpus synthesist is deferred until the corpus has scale, but a lightweight single-repo view earns its keep immediately: an `abcd capture list --dedupe` form renders likely-duplicate captures within one repo at implementation time — the same clustering signal (category, file proximity, similarity) applied to one ledger and shown, rather than synthesised into candidate intents. It is the local, non-agentic subset of this intent, and it pairs with the per-capture form recorded as iss-2608270550201935 (a non-blocking advisory-dedup echo on write) that catches the duplicate one step earlier.

## What's In Scope

- **`/abcd:dredge` command** — cross-corpus synthesis, with sub-verbs (bare invocation = status+help only per the universal convention):
  - `dredge synth` — invoke `issue-synthesist` against open-issue ledgers across all repos in `.abcd/corpus.json`; writes `synthesis-report.{json,md}` to `.abcd/logbook/dredge/<timestamp>/` and surfaces top-N clusters interactively. Flag-shaped modifiers: `--repo <path>` (limit to a single repo), `--since <date>` (only consider issues created/updated since the given date)
  - `dredge promote <cluster-id>` — fold a synthesist cluster into a new intent draft (calls `/abcd:intent new` with the cluster summary + bidirectional links to all member issues; sets `resolves_issues:` reciprocally on the intent)
  - `dredge list` — show recent synthesis reports from `.abcd/logbook/dredge/`
- **`issue-synthesist` agent** — invoked by `/abcd:dredge synth`:
  - Reads open-issue ledgers across all repos in `.abcd/corpus.json`
  - Clusters by category, file proximity, semantic similarity, and frontmatter tags
  - Outputs `synthesis-report.{json,md}` to `.abcd/logbook/dredge/<timestamp>/`
  - Surfaces top-N clusters as candidate intent drafts (operator can promote any via `/abcd:dredge promote <cluster-id>`)
  - Oracle backend chain (same as other agents): host-delegated by default, with the opt-in RepoPrompt adapter and Codex CLI as configured alternatives
- **N:1 promotion semantics** — a cluster of related issues should fold into a single intent. The bidirectional link convention extends `related_intents: [itd-N]` on multiple issues pointing to the same intent, and the intent's frontmatter gains `resolves_issues: [iss-N, iss-NM, iss-NO]` reciprocally. The intent lint extends to verify these reciprocally.
- **Synthesist cadence option** — on-demand via `/abcd:dredge synth`, or periodic via the existing `dev-sync` cadence ([itd-13](itd-13-scheduled-dev-sync.md) covers scheduled `dev-sync`). Default: on-demand; couple to scheduled `dev-sync` if itd-13 has shipped.
- **`intent-fidelity-reviewer` extension** — when an intent ships that was promoted from an issue cluster, the reviewer cross-references whether the cluster's open issues actually moved to `resolved/`. Mismatch = drift finding.

## What's Out of Scope

- **Cross-repo issue copying.** The synthesist *reads* across the corpus to find patterns, but issues stay in the repo where they were captured. No "promote to a global ledger" behaviour.
- **Issue deduplication across repos.** Clustering is a synthesis output, not a destructive merge. Two repos can have overlapping issues; the synthesist surfaces the overlap as a finding, not as a forced unification.
- **Severity-based prioritisation UI.** Severity is frontmatter; no dashboard, no SLA, no triage workflow. The synthesist is the prioritisation surface.
- **Real-time issue tracking integration** (Linear / GitHub Issues sync). The existing `issue-scout` agent already covers GitHub upstream-issue annotation; sync in the other direction is a separate intent.

## Acceptance Criteria

> _BDD format, per `itd-1-acceptance-gates`. These gates are checked by `intent-fidelity-reviewer` when this intent moves to `shipped/`._

- **Given** an abcd-installed repo, **when** the user runs bare `/abcd:dredge`, **then** the command renders status (last-synthesis timestamp + last-known cluster count) plus the full list of previously-synthesised clusters with member issue count, provenance (which corpus repos contributed), and timestamp — per the universal bare-command-as-render discipline (see [`02-constraints/04-naming.md`](../../brief/02-constraints/04-naming.md)). No synthesis runs without an explicit verb.
- **Given** an abcd-installed repo with ≥10 captured issues across the corpus, **when** the user runs `/abcd:dredge synth`, **then** a synthesis report is written to `.abcd/logbook/dredge/<timestamp>/synthesis-report.{json,md}` with at least one identified cluster.
- **Given** a synthesis report with cluster `c-1` containing 3 member issues, **when** the user runs `/abcd:dredge promote c-1`, **then** `/abcd:intent new` is invoked with the cluster summary; the resulting intent has `resolves_issues: [iss-A, iss-B, iss-C]`; each member issue has `related_intents` updated to include the new intent's ID.
- **Given** the corpus has issues only in repo X, **when** the user runs `/abcd:dredge synth --repo X`, **then** clusters are limited to repo X's ledger.
- **Given** the user runs `/abcd:dredge synth --since 2026-01-01`, **when** there are 5 issues created before that date and 3 after, **then** clusters are computed only over the 3 post-date issues.

## Boundary with itd-4 (capture)

itd-4 `/abcd:capture promote` is **strict 1:1** (one issue → one intent). itd-25 `/abcd:dredge promote <cluster-id>` is the **N:1** surface (a cluster of N issues → one intent). The two subverbs do not overlap; the user picks based on whether they're working from one issue or from a synthesis cluster.

## Boundary with itd-36 (memory unification)

itd-36 ships `.abcd/memory/` as the multi-upstream curated knowledge substrate (per the Karpathy LLM Wiki pattern). Dredge synthesis output writes to `.abcd/memory/<type>_<domain>_<slug>.md` with `source.class: dredge_synthesis` (registered in itd-36's source-class enum at [`05-internals/07-memory.md`](../../brief/05-internals/07-memory.md)) — shared destination namespace. The verb stays distinct (per the dredge-pushback in idea-1 R4 review): user moments differ ("look across what we already have" vs "I have a new source"); independent revisit triggers; storage-vs-operation framing prevents surface entanglement. Per-run dredge reports stay at `.abcd/logbook/dredge/<ts>/synthesis-report.{json,md}` per universal pattern 6.

## Revisit triggers

This intent moves from `drafts/` to `planned/` when ANY of the following happens:

1. **First synthesis-worthy ledger volume**: ≥50 issues across the corpus (synthesis below this floor produces noisy clusters).
2. **First user-reported pattern miss**: a user manually identifies a cross-corpus pattern they wished `/abcd:dredge` had surfaced.
3. **itd-13 (scheduled dev-sync) ships**: scheduled sync makes dredge naturally periodic; the coupling decision (auto-couple vs on-demand) becomes concrete.

## Open Questions

- **Synthesist output format.** JSON cluster report is mechanical; the human-readable Markdown should answer "what should I look at first?" — what's the right shape? Top-3 clusters with member-issue links and a one-line "why this matters now" per cluster, or something denser?
- **Wontfix retention.** Do `wontfix/` issues feed the synthesist? Probably yes — patterns of "what we keep deciding not to do" are themselves useful signal. But weight differently from `open/`.
- **Synthesist cadence.** If itd-13 (scheduled dev-sync) hasn't shipped, dredge is on-demand only. If itd-13 has shipped, does dredge auto-couple to dev-sync or stay on-demand?
- **Cluster confidence reporting.** Should each cluster carry a confidence score the reviewer can use to weight findings, or is the user expected to sense-check each cluster manually?

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._
