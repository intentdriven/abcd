---
id: itd-48
slug: intent-fidelity-reviewer-roles-2-3
spec_id: spc-2609211921272106
kind: standalone
suggested_kind: bundle-member
reclassification_history:
  - { date: 2026-05-27, from: "null", to: "standalone", reason: "overrode erroneous capture-time `suggested_kind: bundle-member`; formal bundle-member requires multiple intent files sharing an `spec_id` and `bundle:` id per itd-34, and itd-48 is one intent file." }
supersedes: [itd-31]
related_adrs: []
routed_from: ["spc-33:A1", "spc-33:A2", "spc-33:A3", "spc-33:A4", "spc-33:G1"]
builds_on: [itd-34, itd-5]
severity: major
impact: additive
---

# `intent-fidelity-reviewer` Gains Its Cross-Doc And Kind-Classification Roles

> **Re-scoped on 2026-09-21** by the product thinker: the consistency pass only. The shape role goes to the kinds lint (itd-34) and the overlap question to the pre-pass (itd-42); the headless oracle leg this record named no longer exists. The press release and scope below are read through this paragraph and the Decisions section.


## Press Release

> **abcd's `intent-fidelity-reviewer` agent grows from one `role` to three.** `Role 1` (per-intent fidelity) grades a shipped intent's acceptance criteria against delivered reality and writes `MET`/`MET_WITH_CONCERNS`/`NOT_MET`/`INCONCLUSIVE` verdicts back into the intent's `## Audit Notes`. This intent ships `Role 2` (cross-document consistency — surfaces terminology drift, premise contradictions, scope leakage, sequencing impossibilities, naming conflicts across the brief + intents corpus) and `Role 3` (kind classification — examines whether intents' declared `kind` still fits the corpus and surfaces suggested reclassifications). With all three roles live, `/abcd:intent consistency` and `/abcd:intent shape` move from documented command surface to working command surface. The reviewer becomes the corpus's continuous fidelity auditor.
>
> "The corpus had drifted in three places I hadn't noticed," said Nia, facilitator. "itd-29 still used `epic` throughout its `press release` copy; itd-34's discipline lifecycle said one thing and adr-9's brief amendment said another; itd-37 had been silently bundle-shaped for two intents now without anyone reclassifying. `/abcd:intent consistency` and `/abcd:intent shape` surfaced all three in one pass. None of them were urgent. All of them would have bitten."

## Why This Matters

The `intent-fidelity-reviewer` agent is named in `docs/reference/commands.md` as the engine behind three distinct review verbs on `/abcd:intent`:

| Verb | Role | Status |
|---|---|---|
| `review <itd-N>` | Role 1 — per-intent fidelity | **Shipped (spc-12)** |
| `consistency [<itd-N>]` | Role 2 — cross-doc fidelity | **Documented, not built** |
| `shape [<itd-N>]` | Role 3 — kind classification | **Documented, not built** |

A 2026-05-16 working-log entry named this gap: "No flow-next spec owns `internal/core/lint` or the `intent-fidelity-reviewer` agent" — partially addressed by spc-12 for Role 1, but Roles 2 and 3 still have no owner. spc-12's spec explicitly bounded itself to Role 1; the other two roles "are NOT in scope: they are a later Pass A/B/C agent spec" (spc-12 `## Overview`).

This intent ships those later epics as a **standalone intent whose scope covers both roles** — the two roles share substrate that should land together:

- **Same agent file (`agents/intent-fidelity-reviewer.md`).** Roles 2 and 3 add prompt sections to the same agent, with the same `prompt_version` family, the same injection-canary fixture, the same itd-5 discipline gates. Shipping them separately means two agent CHANGELOG bumps two weeks apart and twice the discipline overhead.
- **Same oracle infrastructure.** Both roles call `_build_cli_oracle()` and follow the same itd-6 cascade ordering. Both ride on itd-47's autonomous-mode oracle fix.
- **Same command-surface conventions.** Both follow the `[<itd-N>]` optional-ID pattern (bare = scan corpus, with ID = scan one intent); both write findings to a structured output the facilitator can act on; both honour the bare-command-as-help convention.

The kind is **`standalone`** per itd-34: a formal `bundle-member` requires multiple intent files sharing a `bundle:` id and an `spec_id`, and itd-48 is one intent file. The "bundle" framing here is metaphorical — two reviewer roles bundled into one spec's scope, not the schema `bundle-member` kind. The capture-time `suggested_kind: bundle-member` advisory was a category error and is preserved as historical advisory data in the frontmatter; `reclassification_history` records the `null → standalone` correction.

The intent is project-agnostic: every abcd project that uses the intent corpus benefits from cross-doc consistency checking and kind reclassification suggestions. The reviewer's behaviour is shaped by the framework, not by any one project's subject matter.

## What's In Scope

### Role 2 — cross-document consistency

- Extend `agents/intent-fidelity-reviewer.md` with a Role 2 prompt section
  that takes the brief + intents corpus as input and surfaces:
  - **Terminology drift** — uses of forbidden synonyms, glossary-undeclared
    terms, or terms used inconsistently across documents.
  - **Premise contradictions** — two intents declaring incompatible
    facts/assumptions about the same surface.
  - **Scope leakage** — an intent's `## What's In Scope` overlapping another
    intent's scope in a way that creates double-coverage or contradictory
    coverage.
  - **Sequencing impossibilities** — intent A claims to depend on intent B,
    but B's scope cannot satisfy A's dependency as written.
  - **Naming conflicts** — same noun used for different concepts; different
    nouns used for the same concept.
- Implement `/abcd:intent consistency [<itd-N>]` routing in
  `commands/intent.md` (the row already exists in the doc but the
  routing is stubbed).
- Findings emitted as structured records grouped by judgement category
  (terminology drift, premise contradictions, scope leakage, sequencing
  impossibilities, naming conflicts) — spc-29 ships finding categories,
  not mechanical lint codes. Mechanical cross-doc categories
  (schema/state contradictions, reference rot, acknowledgement gaps)
  and any associated lint-code namespace are deferred to a follow-up
  intent.
- Reports land at `.abcd/logbook/audit/consistency-<ts>/report.{json,md}`
  (JSON internal, MD render — per the transparency invariant).

### Role 3 — kind classification review

- Extend the same agent with a Role 3 prompt section that examines whether
  intents' declared `kind` still fits the corpus and surfaces suggested
  reclassifications. Inputs: an intent (or all intents) + the corpus the
  intent sits in. Outputs: on scoped runs, a `scoped_verdict.verdict` of
  `KIND_OK` / `KIND_DRIFT` / `INCONCLUSIVE`; on every run, a top-level
  `suggestions[]` list whose entries carry `suggestion_type` ∈
  `{kind_change, bundle, supersession}`. `KIND_DRIFT` pairs with an
  embedded `scoped_verdict.suggestion` that is also reconciled into
  `suggestions[]` (matched by `finding_signature`).
- Implement `/abcd:intent shape [<itd-N>]` routing as an **on-demand**
  verb. Continuous pre-commit shape scanning is **deferred** to a
  follow-up intent — spc-29 ships only the on-demand surface.
- Pairs with `/abcd:intent reclassify` (already documented) — when Role 3
  surfaces a finding, `reclassify` is the action verb that commits it.
- Reports land at `.abcd/logbook/audit/shape-<ts>/report.{json,md}`.
  Findings carry `suggestion_type` ∈
  `{kind_change, bundle, supersession}` with the matching arm fields
  (`current_kind` + `suggested_kind`, `bundle_members`, or
  `superseded_by`); scoped runs additionally emit the
  `KIND_OK` / `KIND_DRIFT` / `INCONCLUSIVE` `scoped_verdict`. spc-29 does
  not introduce a mechanical lint-code namespace for this persona.

### Shared

- Both roles share spc-12's testing pattern: golden fixtures + at least one
  injection-canary per role (per itd-5).
- Both roles call `_build_cli_oracle()` from itd-47's extended version, so
  they run in headless Ralph mode.
- **Pre-commit hook wiring is deferred** to a follow-up intent for both
  roles. spc-29 ships the on-demand verbs (`/abcd:intent consistency` and
  `/abcd:intent shape`) only; the previous draft's promise of a
  `intent-fidelity-consistency` pre-commit hook and Role 3's
  "runs continuously in pre-commit" surface are explicitly out of scope.

## What's Out Of Scope

- **Role 1 modifications.** spc-12 shipped Role 1; this intent does not
  edit it.
- **`lifeboat-oracle` agent.** itd-5's named reviewer; out of scope here.
- **Auto-fix for findings.** Role 2 and Role 3 surface findings; a
  human (or `/abcd:intent reclassify`) acts on them. No
  "auto-apply-suggested-reclassification" verb.
- **Cross-document fidelity beyond the corpus.** External docs (PRs,
  external research) are out of scope; only `brief/` + `intents/` are the
  inputs.

## Scope Conditions

None stated.

## Mechanism

We expect a whole-corpus read to find contradictions no single-record review can, because terminology drift, a premise that two records state differently and a scope that leaked are visible only with both ends in view; shown wrong if a pass over the current corpus finds nothing the ledger did not already hold.

## Acceptance Criteria

- **Given** `abcd intent consistency` runs bare, **when** it completes, **then** a dated report exists under the reviews shelf listing each contradiction found across the brief and every intent (terminology, premise, scope, sequencing, naming), with both ends quoted and located, and the commit it read named.
- **Given** the report's findings, **when** the pass files them, **then** each is an issue naming both ends with the report as its evidence, and a finding the ledger already holds is linked to the existing record rather than filed twice.
- **Given** `abcd intent consistency <itd-N>`, **when** it runs, **then** the pass is scoped to that intent against the corpus and the report says so.
- **Given** the pass has run, **when** the tree is inspected, **then** the brief and the intents are unchanged; the binary assembled the input and validated the return, and the judgement rode the host.

## Decisions

Ruled by the product thinker on 2026-09-21, in the interview that gave this intent its spec:

1. **The consistency pass only.** The shape role is the kinds lint in itd-34; the overlap question is the pre-pass in itd-42.
2. **A report, and a capture per finding**, deduplicated against the ledger.

## Open Questions

_None open; the standalone-versus-two question this record carried is moot with one role left._

## Routed Deferrals (spc-33)

spc-33's Phase 3→4 cleanup sweep routed its cluster-A and G1 deferrals into this
intent (the `routed_from` frontmatter backlinks). This intent shipped the
consistency pass alone and none of the five, so they are tracked by the ledger
record iss-2609260926323349, a future-work seed that names them and hands the
Role-3 item to the kinds lint in itd-34. The list below is what was routed here,
kept as the record of that routing, not as scope this intent delivered:

- **`spc-33:A1`** — Role-2 mechanical half: schema/state contradictions,
  reference rot, acknowledgement gaps → `internal/core/lint --cross-doc` lint codes
  `XD002` / `XD006` / `XD007` (the mechanical cross-doc categories this intent
  defers under "Open Questions → Mechanical Role 2 categories").
- **`spc-33:A2`** — Role-2 pre-commit hook (blocking-vs-advisory policy) — the
  hook wiring this intent lists as deferred under "Shared → Pre-commit hook
  wiring is deferred".
- **`spc-33:A3`** — Role-3 pre-commit scheduling: the `mode="pre_commit"` seam
  exists; the hook wrapper + per-commit-cost policy is the deferred follow-up.
- **`spc-33:A4`** — chunked corpus review. **DORMANT — triggered-by
  `bundle_overflow: true`.** The Role-2 collector / Role-3 classifier set a
  `bundle_overflow` manifest flag on overflow and stop; this item activates ONLY
  when a persisted report actually shows `bundle_overflow: true`. Routing it here
  does NOT make it active implementation scope.
- **`spc-33:G1`** — consistency-report reconciliation key non-convergence: the
  R1 `finding_id` recipe is prose-derived and re-hashes differently across
  re-runs. Fix is structural-key / deterministic-decode / an `addressed` report
  state on `consistency_report.schema.json` — spc-29-owned (this intent owns the
  reviewer + the consistency-report contract).

## Related

- **spc-12** (`intent-fidelity-reviewer` agent — Role 1) — the foundation
  this intent extends, via the shared agent file.
- **itd-47** (autonomous-mode oracle gates) — precondition. Roles 2 and 3
  need headless oracle access; itd-47 ships it.
- **itd-31** (cross-document fidelity reviewer; **superseded** by this
  intent on 2026-05-27 — file moved to
  `intents/superseded/itd-31-cross-document-fidelity-reviewer.md`,
  `superseded_by: itd-48`) — the original intent that named Role 2 as a
  concept; itd-48 absorbs its substance.
- **itd-34** (three intent kinds) — the source of the `kind` taxonomy that
  Role 3 audits.
- **itd-5** (prompt-quality additions discipline) — applies to the agent
  edits in this intent (`prompt_version`, CHANGELOG, injection canaries,
  `capability_scope`).
- **`docs/reference/commands.md`** — the doc whose `consistency` and `shape`
  rows this intent makes real.
- **a dated working-log entry (2026-05-16)** — the gap entry that motivated
  this intent.

## Grounds

- pursued: the autonomous run builds forty-eight intents against this corpus, and a contradiction between two of them is a stop condition it cannot resolve; we expect the first whole-corpus pass to find contradictions the ledger does not hold; shown wrong if it finds none

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-80414ddf96b9 -->
Fidelity review — receipt rcp-80414ddf96b9 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:420fede6b8a9ca127ff19d7173a9d26718dc6e8ee0c6d9a33022efa0451feb9c
Input attestations: diff:internal/core/intent/consistency.go, internal/core/capture/consistency.go, internal/surface/cli/intent_consistency.go, commands/intent.md (Consistency section) at BASE 4ce401c5 (tree 376356ce669156f45b423eb7858a952ffeb0f9d6); live `go run ./cmd/abcd intent consistency --json` on that tree on 2026-09-26@-;

Acceptance rollup: MET 4 · MET_WITH_CONCERNS 0 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: the ingest renders .abcd/work/reviews/< date>-consistency/00-summary.md with review_of_commit pinned to the commit the emit read and one row per finding with both ends quoted and located; TestConsistencyIngestWritesTheDatedReport asserts the path, the pin and the ends, and TestIntentConsistencyRunsEndToEnd drives it through the CLI
  evidence: internal/core/intent/consistency.go:679 — "reportRel := ReviewsShelfRelDir + "/" + dirName + "/00-summary.md""
  evidence: internal/core/intent/consistency.go:1060 — "fmt.Fprintf(&b, "---\nreview_of_commit: %s\n", rv.Commit)"
  evidence: internal/core/intent/consistency.go:277 — "func headCommit(repoRoot string) (string, error)"
  evidence: internal/core/intent/consistency_test.go:276 — "TestConsistencyIngestWritesTheDatedReport"
  evidence: internal/surface/cli/intent_consistency_test.go:79 — "TestIntentConsistencyRunsEndToEnd"
- ac-2 — MET: capture.IngestConsistency files one issue per finding with the report as its evidence and answers Linked: true when an open record already names either end, and the intent core records the link on the row instead of filing twice; TestConsistencyIngestLinksAFindingTheLedgerHolds pins it
  evidence: internal/core/capture/consistency.go:22 — "func IngestConsistency(repoRoot string, payload []byte, date string) (intent.ConsistencyIngestResult, error)"
  evidence: internal/core/capture/consistency.go:37 — "return intent.ConsistencyFiling{IssueID: id, Linked: true}, nil"
  evidence: internal/core/intent/consistency.go:689 — "if filing.Linked {"
  evidence: internal/core/intent/consistency_test.go:330 — "TestConsistencyIngestLinksAFindingTheLedgerHolds"
- ac-3 — MET: assembleConsistency scopes to the named intent and the prompt body says so, the rubric requires every finding to have an end in that intent, and the scoped emit and ingest are tested in core and through the CLI
  evidence: internal/core/intent/consistency.go:315 — "if scope != ConsistencyScopeCorpus {"
  evidence: internal/core/intent/consistency.go:471 — "- scope: %s (%s against the rest of the corpus)"
  evidence: internal/core/intent/consistency.go:111 — "scoped run: when the scope is one intent, every finding has at least one end in that intent"
  evidence: internal/core/intent/consistency_test.go:247 — "TestConsistencyEmitScopesToOneIntent"
  evidence: internal/core/intent/consistency_test.go:354 — "TestConsistencyIngestScopedRun"
  evidence: internal/surface/cli/intent_consistency_test.go:122 — "TestIntentConsistencyScopedAndRefusals"
- ac-4 — MET: the emit assembles the corpus and writes only the local-tier request and corpus files (a live emit on the BASE tree changed nothing under git), the ingest validates the returned findings and refuses with nothing written, the tree-unchanged assertion is in the ingest test, and the judgement is a host pass over the request as the intent-audit shape is
  evidence: internal/core/intent/consistency.go:221 — "fsutil.WriteFileAtomic(filepath.Join(dir, rcp+".request.md"), []byte(doc), 0o644)"
  evidence: internal/core/intent/consistency_test.go:187 — "TestConsistencyEmitAssemblesTheCorpusAndWritesOnlyTheLocalTier"
  evidence: internal/core/intent/consistency_test.go:317 — "AC 4: the brief and the intents are unchanged."
  evidence: internal/core/intent/consistency_test.go:385 — "TestConsistencyIngestRefusesWithNothingWritten"
  evidence: commands/intent.md:868 — "intent consistency --json # the brief and every intent"

Gap audit:
- honoured:
  - the consistency pass (Role 2) is a working command surface, on the CLI and the plugin page
    evidence: internal/surface/cli/intent_consistency.go:26 — "Use: "consistency [< itd-N>]""
    evidence: commands/intent.md:865 — "## Consistency: where do two records contradict each other?"
  - a report and a capture per finding, deduplicated against the ledger (decision 2)
    evidence: internal/core/capture/consistency.go:13 — "with the report as its evidence, and a finding an open record already holds"
  - the report pins the commit it read and marks a dirty corpus
    evidence: internal/core/intent/consistency.go:504 — "- review_of_commit: %s"
    evidence: internal/core/intent/consistency_test.go:590 — "TestConsistencyADirtyCorpusIsMarkedDirty"
- diverged:
  - the press release promised Role 3 (kind classification) and `/abcd:intent shape` as a working surface; decision 1 moved the shape role to itd-34's kinds lint, and no `shape` verb exists on the CLI or the page
    evidence: commands/intent.md:4 — "audit [< itd-N>] | audit --owed [--max < n>] | audit --issue-drift [--strict] | consistency [< itd-N>]"
    evidence: .abcd/development/intents/shipped/itd-48-intent-fidelity-reviewer-roles-2-3.md:52 — "The consistency pass only."
  - the reviews shelf holds no report produced by this pass: the verb shipped without a first whole-corpus run, so the Mechanism's conjecture (the pass finds contradictions the ledger does not hold) is unexercised; the two consistency-named reports on the shelf predate the verb and were hand-written
    evidence: .abcd/work/reviews/2026-07-07-roadmap-consistency/00-summary.md:1 — "# Roadmap-Consistency Review - Consolidated Summary"
    evidence: internal/core/intent/consistency.go:70 — "const ReviewsShelfRelDir = ".abcd/work/reviews""
- missing: (none)
