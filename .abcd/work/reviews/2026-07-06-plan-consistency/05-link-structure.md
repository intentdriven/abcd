# Structural-Integrity Report — documentation corpus

Specialist pass: mechanical link/anchor/index/numbering/naming/orphan checks. Scope scanned: all 224 `.md` files under `.abcd/` and `docs/`, plus root `README.md`, `AGENTS.md`, `CONTRIBUTING.md`, `CHANGELOG.md`. All paths relative to the repo root. Link extraction done mechanically (fenced code blocks and inline code excluded; GitHub-style slugging for anchors).

---

## 1. Broken relative links (70)

**Systemic root causes**, then the full list:

- **(R1) `roadmap/intents/` → `intents/`**: intents moved out of `roadmap/`, many links still say `roadmap/intents/...`.
- **(R2) Wrong lifecycle folder**: intent moved `drafts/`→`shipped/` (etc.) but inbound links not updated.
- **(R3) Vanished trees**: `scripts/`, `.flow/`, `.work/`, `config/`, `tests/`, `agents/`, `.abcd/config.json` do not exist anywhere in this repo (root contains only `cmd/`, `commands/`, `internal/`, `docs/`, `.abcd/`, `.claude-plugin/`, `.githooks/`, `.github/`). The corpus references a prior Python-era repo layout.
- **(R4) Wrong `../` depth** within existing trees.

Full list (`file:line -> target [cause / actual location if it exists]`):

```
.abcd/development/brief/02-constraints/02-dependencies.md:22 -> ../../../scripts/abcd/setup_wizard/README.md            [R3]
.abcd/development/brief/02-constraints/04-naming.md:85 -> ../../../../scripts/abcd/schemas/task_classes.json          [R3]
.abcd/development/brief/03-evidence/04-tradeoffs.md:3 -> ../decisions/adrs/                                            [R4: needs ../../decisions/adrs/]
.abcd/development/brief/04-surfaces/01-ahoy.md:237 -> ../research/notes/ahoy-history-store-manual-scaffolding.md      [R4: needs ../../research/...; file exists]
.abcd/development/brief/04-surfaces/04-launch.md:15 -> ../../../../.flow/specs/fn-64-launch-ship-time-security-gate-fail.md  [R3]
.abcd/development/brief/04-surfaces/04-launch.md:87 -> ../../../config/version-location.json                          [R3]
.abcd/development/brief/04-surfaces/04-launch.md:106 -> ../../../../scripts/abcd/schemas/changelog-entry.schema.json  [R3]
.abcd/development/brief/04-surfaces/07-memory.md:65 -> ../../roadmap/intents/drafts/itd-36-memory-unification.md#acceptance-criteria  [R1+R2: actual ../../intents/shipped/]
.abcd/development/brief/04-surfaces/07-memory.md:114 -> ../../roadmap/intents/drafts/itd-36-memory-unification.md     [R1+R2: actual ../../intents/shipped/]
.abcd/development/brief/05-internals/02-adapters.md:8 -> ../../../../scripts/abcd/README.md                           [R3]
.abcd/development/brief/05-internals/03-configuration.md:100 -> ../../../../scripts/abcd/overlay/README.md            [R3]
.abcd/development/brief/05-internals/03-configuration.md:161 -> ../research/notes/ahoy-history-store-manual-scaffolding.md  [R4: needs ../../research/...]
.abcd/development/brief/05-internals/03-configuration.md:379 -> ../../../../scripts/abcd/README.md                    [R3]
.abcd/development/brief/05-internals/README.md:22 -> ../../../../.work/issues.md                                      [R3: .work/ gitignored/absent; repo convention is now .abcd/.work.local/]
.abcd/development/brief/glossary/README.md:158 -> ../../../config.json                                                [R3: no .abcd/config.json]
.abcd/development/brief/glossary/core/phase.md:25 -> ../../brief/README.md                                            [R4: needs ../../README.md]
.abcd/development/brief/glossary/core/phase.md:28 -> ../../decisions/adrs/adr-9-phase-as-product-layer.md             [R4: needs ../../../decisions/...; file exists]
.abcd/development/brief/glossary/distribution/version.md:20 -> ../../../../config/version-location.json               [R3]
.abcd/development/decisions/adrs/adr-10-phase-negotiator-grounded-tradeoffs.md:49 -> ../roadmap/intents/drafts/itd-41-phase-negotiator.md  [R1+R4: actual ../../intents/drafts/]
.abcd/development/decisions/adrs/adr-15-abstraction-boundary-warn-not-block.md:131 -> ../../roadmap/intents/drafts/itd-52-abstraction-layer-boundary.md  [R1+R2: actual ../../intents/shipped/]
.abcd/development/decisions/adrs/adr-16-fn43-autodrain-boundary-and-gate-defaults.md:140 -> ../../roadmap/intents/drafts/itd-53-review-queue-auto-drain-fidelity-gate.md  [R1+R2: actual ../../intents/shipped/]
.abcd/development/decisions/adrs/adr-16-fn43-autodrain-boundary-and-gate-defaults.md:141 -> ../../../../scripts/abcd/overlay/README.md  [R3]
.abcd/development/decisions/adrs/adr-17-rp-chat-send-override-supersedes-acj1-env-skip.md:157 -> ../../../../.flow/specs/fn-44-rp-chat-send-override-fixed-budget-pre.md  [R3]
.abcd/development/decisions/adrs/adr-17-rp-chat-send-override-supersedes-acj1-env-skip.md:158 -> ../../../../.flow/specs/fn-33-phase-3-to-4-cleanup-placeholder.md  [R3]
.abcd/development/decisions/adrs/adr-17-rp-chat-send-override-supersedes-acj1-env-skip.md:159 -> ../../../../scripts/abcd/overlay/sources/flowctl-dispatcher.sh  [R3]
.abcd/development/decisions/adrs/adr-17-rp-chat-send-override-supersedes-acj1-env-skip.md:160 -> ../../../../scripts/abcd/README.md  [R3]
.abcd/development/decisions/adrs/adr-19-plugin-json-version-carve-out.md:35 -> ../../../config/version-location.json  [R3]
.abcd/development/decisions/adrs/adr-20-manifest-version-lockstep.md:19 -> ../../../config/version-location.json      [R3]
.abcd/development/decisions/adrs/adr-6-rp-review-storage-and-architecture.md:185 -> ../../../../.flow/reviews/README.md  [R3]
.abcd/development/decisions/adrs/adr-6-rp-review-storage-and-architecture.md:187 -> ../../../../<personal account>/abcdZero/docs/development/roadmap/features/planned/F-075-flow-next-local-review.md  [external repo path]
.abcd/development/decisions/adrs/adr-6-rp-review-storage-and-architecture.md:188 -> ../../../../<personal account>/abcdZero/docs/development/roadmap/features/completed/F-037-multi-model-review.md  [external repo path]
.abcd/development/decisions/adrs/adr-7-grill-skill-and-glossary.md:145 -> ../../roadmap/intents/planned/itd-27-grill-skill-and-glossary.md  [R1+R2: actual ../../intents/shipped/]
.abcd/development/decisions/adrs/adr-7-grill-skill-and-glossary.md:147 -> ../../../../tests/fixtures/grill/            [R3]
.abcd/development/decisions/adrs/adr-8-dual-backend-review-asymmetric-trust.md:152 -> ../../roadmap/intents/planned/itd-28-rp-reviews-into-flow.md  [R1+R2: actual ../../intents/shipped/]
.abcd/development/intents/disciplines/itd-37-modification-grammar.md:25 -> ../drafts/itd-36-memory-unification.md     [R2: actual ../shipped/]
.abcd/development/intents/disciplines/itd-37-modification-grammar.md:108 -> ../../../../../.work/idea-assessments/2-programming-as-theory-building.md  [R3]
.abcd/development/intents/disciplines/itd-37-modification-grammar.md:109 -> ../../../../../.work/idea-assessments/3-systems-thinking.md  [R3]
.abcd/development/intents/disciplines/itd-37-modification-grammar.md:114 -> ../drafts/itd-36-memory-unification.md    [R2: actual ../shipped/]
.abcd/development/intents/disciplines/itd-5-prompt-quality-additions.md:152 -> ../../../../../.work/idea-assessments/4-jagged-frontier.md  [R3]
.abcd/development/intents/drafts/itd-25-dredge-cross-corpus-synthesist.md:22 -> itd-4-issue-capture.md                [R2: actual ../shipped/]
.abcd/development/intents/drafts/itd-43-epic-to-spec-terminology.md:39 -> ../../foundation/terminology/               [R3: no foundation/ tree; glossary is at brief/glossary/]
.abcd/development/intents/shipped/itd-34-three-intent-kinds.md:115 -> itd-1-acceptance-gates.md                       [R2: actual ../disciplines/]
.abcd/development/intents/shipped/itd-34-three-intent-kinds.md:115 -> itd-5-prompt-quality-additions.md               [R2: actual ../disciplines/]
.abcd/development/intents/shipped/itd-36-memory-unification.md:109 -> ../../../../../.work/idea-assessments/1-llm-wiki.md  [R3]
.abcd/development/intents/shipped/itd-36-memory-unification.md:116 -> itd-25-dredge-cross-corpus-synthesist.md        [R2: actual ../drafts/]
.abcd/development/intents/shipped/itd-36-memory-unification.md:117 -> itd-26-loot-oss-vendor.md                       [R2: actual ../drafts/]
.abcd/development/intents/shipped/itd-4-issue-capture.md:24 -> itd-25-dredge-cross-corpus-synthesist.md               [R2: actual ../drafts/]
.abcd/development/intents/shipped/itd-4-issue-capture.md:51 -> itd-25-dredge-cross-corpus-synthesist.md               [R2: actual ../drafts/]
.abcd/development/intents/shipped/itd-42-coherence-aware-grill.md:35 -> ../planned/itd-27-grill-skill-and-glossary.md [R2: actual sibling in shipped/]
.abcd/development/intents/shipped/itd-42-coherence-aware-grill.md:45 -> itd-41-phase-negotiator.md                    [R2: actual ../drafts/]
.abcd/development/intents/shipped/itd-42-coherence-aware-grill.md:47 -> itd-39-scope-aware-memory-retrieval.md        [R2: actual ../drafts/]
.abcd/development/intents/shipped/itd-42-coherence-aware-grill.md:61 -> itd-39-scope-aware-memory-retrieval.md        [R2: actual ../drafts/]
.abcd/development/intents/shipped/itd-42-coherence-aware-grill.md:95 -> ../planned/itd-27-grill-skill-and-glossary.md [R2: actual sibling in shipped/]
.abcd/development/intents/shipped/itd-42-coherence-aware-grill.md:96 -> itd-41-phase-negotiator.md                    [R2: actual ../drafts/]
.abcd/development/intents/shipped/itd-42-coherence-aware-grill.md:97 -> itd-39-scope-aware-memory-retrieval.md        [R2: actual ../drafts/]
.abcd/development/intents/superseded/itd-32-audit-role-taxonomy.md:20 -> ../drafts/itd-31-cross-document-fidelity-reviewer.md  [R2: actual sibling in superseded/]
.abcd/development/research/legacy-harvest.md:21 -> ../roadmap/intents/drafts/itd-3-modular-rules-loader.md            [R1+R2: actual ../intents/shipped/]
.abcd/development/research/legacy-harvest.md:32 -> ../roadmap/intents/drafts/itd-24-reflect-command.md                [R1+R2: actual ../intents/planned/]
.abcd/development/research/legacy-harvest.md:91 -> ../roadmap/intents/drafts/itd-24-reflect-command.md                [R1+R2: actual ../intents/planned/]
.abcd/development/research/legacy-harvest.md:211 -> ../roadmap/intents/drafts/itd-3-modular-rules-loader.md           [R1+R2: actual ../intents/shipped/]
.abcd/development/research/notes/fn-34-flowctl-divergence-audit.md:70 -> ../../../.flow/specs/fn-34-detach-scriptsralphflowctlpy-from.md  [R3]
.abcd/development/research/prompting/agents/chat-distiller.md:111 -> ../../../../agents/chat-distiller.md             [R3: no agents/ tree]
.abcd/development/research/prompting/agents/chat-distiller.md:113 -> ../../roadmap/intents/drafts/itd-11-pass-b-pitfall-mitigation.md  [R1+R4: actual ../../../intents/drafts/]
.abcd/development/research/prompting/agents/chat-distiller.md:114 -> ../../roadmap/intents/drafts/itd-5-prompt-quality-additions.md  [R1+R2+R4: actual ../../../intents/disciplines/]
.abcd/development/research/prompting/agents/embark-scaffolder.md:111 -> ../../../../agents/embark-scaffolder.md       [R3]
.abcd/development/research/prompting/agents/embark-scaffolder.md:113 -> ../../roadmap/intents/drafts/itd-16-hash-chain-merkle-audit.md  [R1+R4: actual ../../../intents/drafts/]
.abcd/development/research/prompting/agents/embark-scaffolder.md:114 -> ../../roadmap/intents/drafts/itd-5-prompt-quality-additions.md  [R1+R2+R4: actual ../../../intents/disciplines/]
.abcd/development/research/prompting/agents/embark-scaffolder.md:115 -> ../../brief/README.md                         [R4: needs ../../../brief/README.md]
.abcd/development/roadmap/README.md:123 -> ../activity/                                                               [R3: no activity/ dir]
```

Root files (`README.md`, `AGENTS.md`, `CONTRIBUTING.md`, `CHANGELOG.md`): **all relative links resolve**.

### 1b. Broken anchor fragments (20) — target file exists, anchor doesn't

All verified against actual headings; cause noted:

| Link | Cause |
|---|---|
| `brief/01-product/01-press-release.md:31 -> ../04-surfaces/01-ahoy.md#1-acceptance` | Section renamed/renumbered — heading is now unnumbered `## Acceptance` (slug `acceptance`) |
| `brief/06-delivery/01-build-sequence.md:48 -> ../04-surfaces/01-ahoy.md#1-acceptance` | Same |
| `brief/01-product/02-context.md:37`, `02-constraints/01-platform.md:17`, `02-constraints/03-invariants.md:19`, `04-surfaces/02-disembark.md:147`, `05-internals/04-universal-patterns.md:121` `-> 03-embark.md#7-voyage-layout-embarkdisembark-provenance-and-history` | Heading is `## 7. Voyage layout — embark/disembark provenance and history` → slug `7-voyage-layout--embarkdisembark-provenance-and-history` (double hyphen from em-dash) |
| `brief/04-surfaces/03-embark.md:26,46,93 -> #7-voyage-layout-…` (intra-file, ×3) | Same stale slug within its own file |
| `brief/02-constraints/01-platform.md:11 -> ../01-product/02-context.md#repos` | `02-context.md` has no headings besides `# Context` — section gone |
| `brief/02-constraints/02-dependencies.md:26`, `03-invariants.md:13` `-> ../05-internals/04-universal-patterns.md#3-plugin-preferred-internal-fallback` | Heading is `## 3. Plugin-preferred + internal-fallback` → slug `3-plugin-preferred--internal-fallback` |
| `brief/04-surfaces/02-disembark.md:5`, `03-embark.md:5` `-> ../01-product/03-mental-model.md#the-naurian-gap` | Heading is `## The Naurian gap — Modification axis` → slug `the-naurian-gap--modification-axis` |
| `brief/04-surfaces/README.md:11`, `05-internals/01-agents.md:20,32` `-> 05-intent.md#6-the-intent-fidelity-reviewer-agent-three-roles-three-verbs` | Section renumbered 6→7 (`## 7. The intent-fidelity-reviewer agent…`) |
| `brief/05-internals/06-lint.md:48 -> ../04-surfaces/05-intent.md#5-acceptance-gates-and-bidirectional-link-verification` | Section renumbered 5→6 |
| `intents/disciplines/itd-1-acceptance-gates.md:76 -> …05-intent.md#6-the-intent-fidelity-reviewer-agent-three-roles` | Truncated slug AND renumbered 6→7 |

### 1c. Link-text vs href drift (resolves, but label lies)

- `.abcd/development/decisions/README.md:150` — text says `../roadmap/intents/`, href is `../intents` (correct).
- `.abcd/development/intents/README.md:197,408` — text says `../phases/`, href is `../roadmap/phases` (correct).
- `.abcd/development/research/notes/README.md` ("Related") — text `../roadmap/intents/`, href `../../intents`.
- `.abcd/development/roadmap/rfcs/README.md:48` — prose asserts "Intents live at `.abcd/development/roadmap/intents/`" — false.
- `.abcd/development/roadmap/README.md:72` — the "stale-proof" intent-count shell snippet uses `ls .abcd/development/roadmap/intents/$b/itd-*.md` — wrong path; prints 0 for every bucket. (Ironic given the snippet exists to avoid drift.)

---

## 2. README indexes vs reality

### `.abcd/development/intents/README.md` — severely stale (worst file in corpus)

- **Drafts listing (lines 256–296)** — listed but NOT in `drafts/`: `itd-2` (→shipped/), `itd-3` (→shipped/), `itd-4` (→shipped/), `itd-20` (→planned/), `itd-24` (→planned/), `itd-34` (→shipped/), `itd-36` (→shipped/), `itd-40` (→shipped/). On disk in `drafts/` but NOT listed (15): `itd-43, itd-44, itd-45, itd-50, itd-51, itd-54, itd-55, itd-56, itd-57, itd-59, itd-60, itd-61, itd-62, itd-64, itd-70`.
- **Planned listing (lines 319–325)** — says "One intent currently planned: itd-6"; `itd-6` is actually in `shipped/`. On disk in `planned/` but NOT listed (8): `itd-20, itd-24, itd-63, itd-65, itd-66, itd-67, itd-69, itd-72`.
- **Shipped listing (line 355ff)** — lists only `itd-27, itd-28`. On disk in `shipped/` but NOT listed (15): `itd-2, itd-3, itd-4, itd-6, itd-34, itd-36, itd-40, itd-42, itd-46, itd-47, itd-48, itd-49, itd-52, itd-53, itd-58`.
- **Disciplines (302ff)**: `itd-1, itd-5, itd-37` — matches disk exactly. ✓
- **Superseded (334ff)**: `itd-31, itd-32` — matches disk exactly. ✓

### `.abcd/development/brief/glossary/README.md`

- Directory-layout block labels the root `terminology/` — the directory is named `glossary/`.
- Layout + Term Index omit the entire `distribution/` context (`end-user.md`, `release.md`, `version.md` on disk, unindexed) and `core/disembark.md` (on disk, unindexed).
- Line 22: "Current contexts: `core`, `interview`" — omits `distribution`.
- All indexed entries do have files. ✓
- Prose references nonexistent paths: `scripts/abcd/schemas/terminology.schema.json`, `scripts/abcd/lint_terminology.py`, `.abcd/development/foundation/terminology/`, `skills/abcd-intent-grill/phase-1-glossary-mode.md`, `tests/abcd/test_issue_schema.py`, `.abcd/config.json`.

### `.abcd/development/decisions/adrs/README.md`

- Index table lists adr-1…adr-20; disk has exactly adr-1…adr-20. **Exact match.** ✓

### `.abcd/development/research/prompting/agents/README.md`

- Inventory row (line 49) marks `intent-fidelity-reviewer` as **TBD**, but `intent-fidelity-reviewer.md` exists on disk in the same directory.
- Header says "the 14 abcd agents" / "Inventory (target: 14 files)" but the table has 15 rows, and the brief (`05-internals/README.md:7`, `01-press-release.md:19`, `04-scope.md:25`, `05-prompt-quality.md:3`, `03-configuration.md:346`) says **15 agents**. `research/prompting/01-general-best-practices.md` internally mixes "15 agents" (lines 5, 38, 117) and "14 abcd agents"/"14-agent" (lines 30, 135).

### `.abcd/development/brief/README.md`

- Navigation table and directory-layout tree omit `glossary/` (and `glossary/_template.md`) entirely, though it lives under `brief/`. (`development/README.md` does mention it.)
- Chapters 01–06 listed all exist. ✓

### Exact matches ✓
- `brief/04-surfaces/README.md` — 9 rows ↔ 9 files 01–09.
- `brief/05-internals/README.md` — 10 rows ↔ 10 files 01–10.
- `roadmap/phases/README.md` — phase-0…phase-5, disk matches.

### Others
- `roadmap/rfcs/README.md` — has no index of actual RFCs; `rfc-1-pirate-mode-yolo-for-power-users.md` exists on disk, mentioned only obliquely with no link.
- `decisions/notes/README.md` — purpose-only, no index; 8 `fn-*` note files on disk unindexed and unlinked from anywhere.
- `research/notes/README.md` — purpose README with examples (all example files exist ✓), but routes "MCP/architecture contracts" to `research/adr/` and "phase-scoped research" to `research/phase/<N>/` — **neither directory exists**; the MCPBridge contract files sit in `notes/` itself, violating the README's own routing. Its "Related" line names sibling dirs `phase/`, `prompting/`, `adr/` — only `prompting/` exists (plus unmentioned `spikes/`).
- `development/README.md` — says "ADRs use sequential `NNNN`" (actual: unpadded `adr-N`); says "plans and research notes are date-prefixed" — no research note is date-prefixed, and `research/notes/README.md` prescribes `<topic>-<kind>.md` instead. Also says "Issues graduate into `intents/` or `principles/` rather than a ledger", while several brief/ADR files still link a `.work/issues.md` ledger (see §1).
- `docs/` READMEs — internally consistent; all four subdirectory links resolve; `docs/reference/cli/README.md`'s pointer to `internal/surface/cli/` is valid. Note: every `docs/` subtree contains only its README (no content pages yet).

## 3. Numbering integrity

- **Brief chapters**: `00-meta.md`, `01-product/` … `06-delivery/` — sequential, no gaps/duplicates. Files inside every chapter are sequential from 01 with no gaps or duplicates. ✓
- **ADRs**: adr-1…adr-20 — **no gaps, no duplicates**. ✓
- **Intents (itd-N)**: present 1–72 except **gaps: itd-38, itd-68, itd-71**. (itd-38 documented as released; itd-68/71 appear nowhere in any lifecycle folder; no file documents the retirement — contrast itd-31/32 which are preserved in `superseded/`.) **No number appears in two lifecycle folders at once.** ✓
- **RFCs**: rfc-1 only. ✓
- **Phases**: phase-0…phase-5, sequential. ✓
- **research/notes**: `01-`, `02-`, `03-` numeric prefixes on the three harness/MCPBridge notes contradict the directory's own stated convention ("Notes are not a numbered record-type"); `fn-25-closeout/` uses a third scheme (`t1-*`).

## 4. File naming consistency

- **kebab-case**: universal, with these deviations (all deliberate-looking but unexplained locally except where noted):
  - `.abcd/work/CONTEXT.md`, `.abcd/work/DECISIONS.md` — UPPERCASE; convention explained in root `AGENTS.md` and `.abcd/README.md`. ✓
  - `_template.md` ×2 (`brief/glossary/`, `research/prompting/agents/`) and `_references.md` (`research/`) — leading-underscore convention; glossary README explains its `_template.md`; `_references.md` is explained nowhere.
  - `mcpbridge_probe.py` — snake_case (Python convention; acceptable for a spike).
- **Prefix style**: uniformly lowercase unpadded `adr-N-slug` / `itd-N-slug` / `rfc-N-slug` / `phase-N-slug` / two-digit `NN-` in brief. **No `ADR-NNN` style mixing anywhere on disk.** The only padded-style claim is the stale "sequential `NNNN`" line in `development/README.md` (§2).
- Directory naming: singular/plural mix follows the parent standard.

## 5. Orphans, stubs, strays

**Orphaned .md files** (zero inbound links from any in-scope file or root README/AGENTS/CONTRIBUTING/CHANGELOG; READMEs excluded; no file outside scope links into `.abcd/` either — verified by grep over `commands/`, `internal/`, `cmd/`, `.claude-plugin/`, `.github/`): 92 files total. Largest classes:

- **56 intent files** — every intent in `drafts/` (all 39), `planned/` (7 of 8), `shipped/` (17 of 17… minus the few reached only via broken links), `superseded/itd-32` — orphaned largely *because* the `intents/README.md` listings are stale (§2) and inbound links point at wrong lifecycle folders (§1/R2). Structurally these are "indexed collections" whose index is broken.
- **All 8 `decisions/notes/fn-82-*` / `fn-59-*` files** — README explains the directory's purpose but indexes nothing.
- **15 research notes** (e.g. `abcd-lineage.md`, `predecessor-notes.md`, `transcript-sampling.md`, `spike-mcp-evidence.md`, `fn-25-closeout/t1-flowctl-validate-integration.md`) — README describes the genre, links none.
- `roadmap/rfcs/rfc-1-pirate-mode-yolo-for-power-users.md` — unindexed (§2).
- `brief/03-evidence/01–04` (all four), `01-product/01-press-release.md`, `02-constraints/02,03` — chapter dirs have no README and the parent README links folders, not files.
- `brief/glossary/_template.md`, `glossary/core/disembark.md`, `glossary/distribution/end-user.md` — the latter two are exactly the glossary-index omissions (§2).
- `research/prompting/agents/_template.md` (referenced in prose, never linked), `agents/intent-fidelity-reviewer.md` (the "TBD" row, §2).
- `.abcd/work/CONTEXT.md`, `.abcd/work/DECISIONS.md` — explained by `AGENTS.md`/`.abcd/README.md` prose, but never linked.

**Stub files (<5 non-empty lines)**: only `brief/01-product/05-personas.md` (4 lines — dense and complete, not a placeholder; references `personas.py`, which doesn't exist in this Go repo — another R3-class stale reference).

**`.gitkeep` in non-empty dir**: `.abcd/development/intents/superseded/.gitkeep` — stale; the directory has contained `itd-31`/`itd-32` files. No other `.gitkeep` in scope.

**Stray non-markdown files**: `.abcd/development/personas.json` (explained by `development/README.md` ✓); `.abcd/development/research/spikes/mcpbridge_probe.py` (a Python spike in a Go repo; `development/README.md` explains spikes generically, but `research/spikes/` itself has no README). No `.DS_Store` anywhere.

**Directories with no README** (parent standard requires one per non-trivial dir): `.abcd/work/`, `research/`, `research/prompting/`, `research/spikes/`, `research/notes/fn-25-closeout/`, `brief/01-product/`, `brief/02-constraints/`, `brief/03-evidence/`, `brief/06-delivery/`, `brief/glossary/{core,distribution,interview}/`, and all five `intents/{drafts,planned,shipped,disciplines,superseded}/` (brief/README explicitly acknowledges "where present" for its chapters; the rest are unacknowledged).

## 6. Symlink and case issues

- `CLAUDE.md -> AGENTS.md` symlink **intact**, target exists and is non-empty. Bonus: `GEMINI.md -> AGENTS.md` also intact.
- **No case-mismatch links** found anywhere (every resolving link matches on-disk casing component-by-component).

---

### Highest-leverage fixes (by blast radius)

1. Global `roadmap/intents/` → `intents/` link sweep (fixes ~15 broken links at once).
2. Regenerate the five lifecycle listings in `intents/README.md` from disk (fixes the worst index and explains ~56 "orphans").
3. Re-slug the four renamed/renumbered `05-intent.md` / `03-embark.md` / `01-ahoy.md` / `03-mental-model.md` anchors (fixes 18 of 20 anchor breaks).
4. Decide policy for R3 references (`scripts/`, `.flow/`, `.work/`, `config/`, `tests/`, `agents/`) — they describe a predecessor repo layout absent from this Go repo; either port the artifacts or rewrite the ~30 references.
5. Delete stale `intents/superseded/.gitkeep`; add glossary `distribution/` context + `disembark` to the glossary index; flip the `intent-fidelity-reviewer` TBD; settle 14-vs-15 agent count.
