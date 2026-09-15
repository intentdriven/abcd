# Tradeoffs Considered

> **Status: PARTIAL.** Tradeoff-shaped content (decisions made + alternatives weighed, with rationale) accumulates here as the project progresses. Inflection-point decisions also live in [`../../decisions/adrs/`](../../decisions/adrs/) — this file is the lightweight rolodex, the ADRs are the deep dive.

> **Historical evidence — the architecture has moved on.** Several tradeoffs recorded below predate the rebuild and describe mechanisms the new architecture overturns — notably the fixed oracle cascade (RP → Codex → in-session) and the multi-pass agent pipeline framing. The oracle is now **host-delegated by default** with opt-in adapters ([adr-25](../../decisions/adrs/0025-host-delegated-llm-default.md)), and the bundled tools those tradeoffs assumed are now **pluggable adapters over native defaults** ([adr-22](../../decisions/adrs/0022-bundled-deps-as-pluggable-adapters.md)). This content is **kept as evidence** of the decisions made and the alternatives weighed at the time — read the "Reconsider when" lines against the current ADRs, not as live design.

## Purpose

This file lists architectural tradeoffs the team considered explicitly: "we chose X over Y because Z". The next agent reading this file should understand *not just what was chosen*, but *what alternatives were considered and why they lost*. This protects against cargo-cult ("rebuild what existed") and against re-litigation ("propose alternative Y again without knowing why it was rejected").

## Format

For each entry:

```markdown
## <Decision name>

- **Chose:** <option>
- **Over:** <one or more alternatives>
- **Why:** <the property that made the chosen option win — usually a constraint, a simplicity argument, or evidence from a prior failure>
- **Reconsider when:** <conditions under which the rejected alternatives might win — e.g., scale, new tooling, platform shift>
- **Source:** <ADR / commit / review thread>
```

## Tradeoffs recorded

### `--resume` flag removed from `/abcd:disembark`

- **Chose:** no resume sub-verb, and no checkpoint state to resume from. A pack is a single deterministic pass that either completes or writes nothing, and no run writes a `_state.json` at any path. The only run-level record is the operator-local voyage log, `~/.abcd/voyage/<source-root-sha>/disembark/history.jsonl`, appended after a pack lands.
- **Over:** earlier drafts had a `--resume` flag that would re-attempt a previously failed disembark from the last checkpoint.
- **Why:** Resume semantics across the three-pass agent pipeline (Pass A spine + Pass B chat-distill + Pass C synthesis) require partial-output handling, agent-state restoration, and idempotency guarantees that the current release doesn't yet have. Re-running `disembark pack <source-repo> <dest>` from scratch is fast enough at current corpus sizes that resume is overhead, not value.
- **Reconsider when:** disembark runs become long enough on real-world corpora that re-running from scratch is noticeably expensive AND a clear restart-after-failure user moment surfaces (autonomous overnight runs hitting rate limits; CI integration where re-running is expensive).
- **Source:** [`04-surfaces/02-disembark.md`](../04-surfaces/02-disembark.md), which carries no resume sub-verb and no checkpoint file; logged in the `.abcd/work/issues/` ledger (P1 #5) on 2026-05-08.

### Memory unification (idea-1 → itd-36)

- **Chose:** Widen `.abcd/memory/` to multi-upstream curated substrate (session memory + external sources + reviews + notes + dredge synthesis); ship `/abcd:memory` command initially with `ingest`/`ask`/`lint` sub-verbs; default-no-originals + `--keep-original` opt-in; per-page + cumulative quotation lints.
- **Over:** New separate `.abcd/knowledge/` namespace (initial framing); pushing to a later phase.
- **Why:** Initial proposal was a new namespace, but RP review revealed that the visibility-rule lock ("no exceptions") forbids per-subdirectory carve-outs, and `.abcd/memory/` already exists as the per-project compounding-curated knowledge layer (just narrowly scoped to session memory). Widening the upstream funnel of an existing namespace is structurally smaller than creating a new one. User reframed at R3; substrate reuse with itd-26 (loot) compressed the initial cost shape further.
- **Reconsider when:** the worked-example test (3 adversarial walkthroughs across research-shaped, tooling, mixed-licence projects) shows sprawl or licence-gate failure — answer reverts to a later phase with the failed example as primary debug.
- **Source:** `.abcd/work/idea-assessments/1-llm-wiki.md`; chat `adversarial-review-llm-w-4970CE` (5 rounds).

### Modification grammar discipline (idea-2 → itd-37); Ripple axis absorbing idea-3

- **Chose:** the itd-37 modification-grammar discipline with a 4-axis `## Modification Grammar` section per spec (`Extends cleanly` / `Breaks the design` / `Why` / `Ripple`); two-page-class extraction. idea-3's per-spec-external "system impact" half folds into itd-37 as the Ripple axis; itd-38 ID released (not reserved).
- **Over:** Two separate disciplines (itd-37 internal + itd-38 external); standalone `system-impact awareness` discipline; reserve-and-rename for itd-38; new `.abcd/development/system-state/` namespace.
- **Why:** RP review (idea-3) judged the two-discipline split rhetorical not architectural — system-impact and modification-grammar pages for the same spec always sit beside each other, get curated together; same retrieval key (domain). Cost-benefit: 4 disciplines per spec is meaningfully cheaper than 5, and the per-spec authoring cost grows roughly linearly with discipline count. itd-38 had three candidate identities (discipline-set audit / system-state-registry / emergent-pattern-emission-rules) which is three intents not three names; releasing the ID prevents arbitrary stapling.
- **Reconsider when:** observed evidence shows a `## Ripple` axis wants to be a separate page-class (pre-empted by the collapse) OR a discipline-set audit trigger fires (≥5 disciplines OR observed contradiction).
- **Source:** `.abcd/work/idea-assessments/3-systems-thinking.md`; chat `idea-3-itd-38-adversaria-31A06A` (2 rounds).

### Jagged frontier — split static (itd-5, current) vs dynamic (Frontier Awareness, a later phase)

- **Chose:** the itd-5 extension adds `capability_scope` static frontmatter (same artefact class as `prompt_version`; cheap one-shot frontmatter addition per agent at v1.0.0 lock); a later-phase standalone Frontier Awareness intent owns `known_failure_modes` runtime events + plan-time semantic check + capability-aware **pre-cascade selector** (layer above the cascade, NOT modification of itd-2/itd-6 contract); `/abcd:frontier` command (bare = render; sub-verbs `flag`/`history`/`explain`).
- **Over:** Single current-release discipline (itd-5 + dynamic half conflated), or single later-phase standalone (everything dynamic), or modifying the cascade contract directly.
- **Why:** RP review (idea-4) flagged that conflating static + dynamic in itd-5 silently moves itd-5 from cheap to expensive; that the cascade-considers-capability framing silently reopens itd-2 + itd-6 (which currently say "fixed cascade" in `04-universal-patterns.md § 7`); and that `.abcd/toolchain-state/` was the third namespace-creep proposal in three reviews. Splitting static (cheap, current) vs dynamic (expensive, a later phase) preserves the cost discipline; pre-cascade selector framing preserves the cascade contract.
- **Reconsider when:** itd-5 ships and the static `capability_scope` data accumulates across 10+ agents, providing the prior for the later-phase Frontier Awareness dynamic check.
- **Source:** `.abcd/work/idea-assessments/4-jagged-frontier.md`; chat `idea-4-jagged-frontier-r-33C475` (2 rounds).

### Bare-command-as-render discipline (sweep)

- **Chose:** Every `/abcd:<verb>` command treats bare invocation (no args) as **status + help + render of current state**. Sub-verbs MUST earn their existence by mutating, taking a positional arg, scoping to a different time-axis or granularity, or performing an action distinct from rendering. `<verb> show`, `<verb> stats`, `<verb> list` (plain), `<verb> view` are forbidden — they name what bare already does. Lint code `SD001`.
- **Over:** Per-command bare conventions (some commands had `show` / `stats` / `list` sub-verbs imported from CLI conventions like `git status`, `npm list`, `abcd oracle stats`).
- **Why:** Surface drift was a systemic failure mode (8 violator sites across 7 files surfaced in one audit). The discipline rules `show`/`stats`/`list`/`view` out of the namespace at design time. Bare-as-render is what gives abcd its discoverability ("type the verb, see what it does"); sub-verbs that just rename "show me the state" obscure the discoverability instead of enhancing it. Earned exception preserved: `/abcd:capture list --open|--resolved|--wontfix|--all` (filtered query distinct from default-open render).
- **Reconsider when:** a sub-verb shape genuinely earns its existence (mutates state OR takes positional arg OR distinct time-axis/granularity OR distinct action) and the bare convention can't carry it.
- **Source:** `02-constraints/04-naming.md § Bare-command-as-render discipline`; logged in the `.abcd/work/issues/` ledger on 2026-05-08; sweep landed across 8 files in same session.

## Why this is separate from ADRs

Architecture Decision Records ([`.abcd/development/decisions/adrs/`](../../decisions/adrs/)) are full multi-page records of architectural decisions, with stakeholders, alternatives evaluated, and consequences. This file is the **lightweight rolodex** of tradeoff one-liners — a quick reference for "did we already think about this?". ADRs are the deep dive; this file is the index.
