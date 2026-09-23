---
id: itd-95
shipped_in: v0.4.0
slug: disembark-grounds-a-lifeboat-s-open-questions-on-a-repo-s-to
spec_id: spc-12
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: additive
---

# Disembark Grounds A Lifeboat's Open Questions On A Repo's TODO And FIXME Markers

## Press Release

> **A lifeboat's "open questions" no longer comes back empty just because the project never kept an abcd record.** When `abcd disembark` probes a repository, it reads the open questions a team left in the one place every codebase keeps them — the `TODO` and `FIXME` markers scattered through the source. A conventions-tier adapter scans the working tree for those markers and grounds the `evidence/open-questions` brief section against them, citing the files it found. A repository with no `.abcd/` record, but a codebase full of "TODO: handle the retry case", now packs a lifeboat whose open-questions section reflects what the code itself admits is unfinished — instead of a blank a rescuer has to fill from nothing.
>
> "I pointed disembark at an abandoned service with no notes and a hundred TODOs in it," said Alice, who was handed the wreck to revive. "The lifeboat used to tell me 'no open questions on record' — which was a lie the moment I opened the source. Now it hands me the actual list the last team scribbled to themselves. That is exactly the thread I need to pull."

## Why This Matters

Disembark grounds each brief section through tiered `Source` adapters (`internal/core/lifeboat/`): git history (Tier-0), conventional project files (Tier-1), and an abcd record (Tier-2 native). The richest tier present wins, and every section that nothing can ground comes back as a first-class **blank** — a named question a human must answer (adr-35).

Today `evidence/open-questions` is grounded by exactly one adapter: `nativeOpenQuestionsSource`, which counts open issues and intents in an `.abcd/` record. There is no git-tier or conventions-tier adapter for it. So on any repository without a native record — the common case for a rescue — the section falls straight through to blank, **even when the code is dense with `TODO`, `FIXME`, and `XXX` markers**. The mapping contract (`internal/core/lifeboat/mapping.go`) already anticipates this: the `evidence/open-questions` row lists "TODO and FIXME markers, open issues, intents' open-questions sections" as what it reads, and predicts a *partial* status at the conventions tier — a prediction no adapter yet delivers. The blank is therefore not a human-owned question; `evidence/open-questions` is a `KindExtractable` section (adr-36), so an unfilled blank here is **coverage debt abcd can close with a better adapter**, exactly the kind of gap the M2 cross-repo gate probe exists to surface.

This matters because open questions are among the highest-value things a lifeboat can carry. A rescuer inheriting a dead project needs to know what the previous team knew was unfinished — and in a project that never kept a record, the source markers are the only surviving trace of it. Closing this gap turns a systematically empty section into a grounded one for the majority of real-world source repositories.

Promoted from the open ledger issue iss-99.

## What's In Scope

- **A conventions-tier `Source` adapter for `evidence/open-questions`.** It scans the source tree for in-code work markers (`TODO`, `FIXME`, and similar — the exact set is an open question below) and grounds the section against them, citing the files where markers were found. It implements the same `Source` interface every other adapter does (`Section()` / `Tier()` / `Probe()`), reads only through the read-only `SourceContext` surface, and never writes to the source repository.
- **Honest three-valued status.** Markers found → a non-blank status (grounded or partial — see open questions) with cited evidence. No markers found → a blank carrying what was searched and the human question, consistent with every other adapter's blank contract.
- **Bounded, safe scanning.** The scan respects the probe's existing safety envelope — the per-file read cap (`maxProbeReadBytes`), directory-entry cap (`maxDirEntries`), and containment root — and adds whatever additional caps a whole-tree marker scan needs so a hostile or vast repository cannot exhaust memory or time. The precise caps are an open question.
- **A coverage-report row that now grounds where it used to blank.** Running `abcd disembark probe` against a marker-bearing, record-less repository reports `evidence/open-questions` as non-blank with the markers as evidence — the visible readout of the closed gap.

## What's Out of Scope

- **Interpreting or resolving the markers.** The adapter reports the markers as evidence; it does not judge which are stale, cluster them, or turn them into intents. Distillation of open questions into designed work is downstream (the synthesis passes and the press-release composer), not this adapter.
- **Changing the tier-reduction model.** This intent adds one adapter; it does not change the rule that the richest present tier wins a section. If a repository has both an abcd record and TODO markers, the reduction behaviour is called out as an open question, not redesigned here.
- **A git-history marker archaeology.** Reading markers added and removed across history (a Tier-0 signal) is a separate, richer idea; this intent scans the current working tree only.
- **New marker-scanning dependencies.** The adapter reuses abcd's own read surface; adopting an external scanner would be a dependency decision (see SOTA).
- **The other missing conventions adapters** (`constraints/naming`, `internals`) — those are iss-100 / itd-96, a sibling intent.

## Scope Conditions

None stated.

## Acceptance Criteria

> _BDD format, per the [itd-1 discipline](../disciplines/itd-1-acceptance-gates.md). Some bars below reference decisions still open in § Open Questions; they are written to hold whichever way those resolve._

- **Given** a git repository with no `.abcd/` record whose source carries in-code work markers (`TODO` / `FIXME` / …), **when** `abcd disembark probe` runs against it, **then** `evidence/open-questions` is reported non-blank, cites the file(s) where markers were found, and is no longer a blank.
- **Given** a repository with no record and no work markers anywhere in its tree, **when** the probe runs, **then** `evidence/open-questions` comes back blank with a populated `Searched` list and a human `Question`, exactly as the blank contract requires — a marker scan that finds nothing is an honest blank, not a fabricated result.
- **Given** any source repository, **when** the marker adapter's `Probe` runs, **then** the source tree is byte-for-byte identical afterwards — the adapter is read-only by construction and touches nothing outside the containment root.
- **Given** a pathologically large or hostile tree (very many files, an enormous single file), **when** the adapter scans it, **then** the scan stays within the probe's declared caps and returns rather than exhausting memory or hanging.
- **Given** a repository that has both an abcd record grounding `evidence/open-questions` and TODO markers in its source, **when** the probe runs, **then** the section resolves to a single defined result per the documented tier-reduction rule (the chosen behaviour is fixed by an open question below, but the outcome is deterministic and explained in the coverage report, never ambiguous).

## Prior Art

- **`internal/core/lifeboat/sources_conventions.go`** — the conventions-tier adapter pattern this intent extends: each `conv*Source` reads a section through the `SourceContext` file surface and never touches git. The new adapter is a sibling of these.
- **`internal/core/lifeboat/sources_native.go` — `nativeOpenQuestionsSource`** — the only adapter grounding `evidence/open-questions` today, from open issues and intents. The new adapter grounds the same section one tier down, for repositories with no record.
- **`internal/core/lifeboat/mapping.go`** — the brief-to-lifeboat contract. The `evidence/open-questions` row already names "TODO and FIXME markers" as a read and predicts a *partial* conventions status; this intent delivers the adapter that makes the prediction true.
- **[adr-35](../../decisions/adrs/0035-lifeboat-as-coverage-experiment.md)** (lifeboat as a coverage experiment) — a blank is a first-class result; this intent turns a systematically empty section grounded.
- **adr-36** (extractable vs human-owned sections) — `evidence/open-questions` is `KindExtractable`, so its blank is coverage debt to close, which is what this intent does. Promoted from the open ledger issue iss-99 (found during the M2 cross-repo gate probe).
- **[reality-is-filable](../../principles/reality-is-filable.md)** — the markers are a fact in the tree; grounding a section on them records reality rather than asking a human to reconstruct it.

## SOTA

> _Per the [sota-per-intent principle](../../principles/sota-per-intent.md): existing alternatives + rough maturity, then a chosen path. First-pass sketch for a draft; the confirmation is a pre-build gate._

- **In-code marker scanning.** Alternatives: dedicated TODO/FIXME scanners and reporters (`leasot`, `todocheck`, `todo-tree`, `git grep`, ripgrep patterns, and "TODO → issue" CI bots) — *mature* as standalone tools, but each is a whole CLI or CI action that emits its own report format and would be an external dependency, not a drop-in for abcd's tiered `Source` / `Evidence` contract. None grounds a brief section or speaks the coverage report's shape.
- **Chosen path — Path 2 (basic-with-seam).** A small native marker scan, built on the existing read-only `SourceContext` surface and returning `Evidence` like every other adapter. No new dependency. The seam is the `Source` interface itself: a richer external scanner could later sit behind the same interface for this section without changing the reduction model.
- **Verdict — Path 2.** No new dependency ⇒ no gate-1 stop; the seam (the `Source` interface) is real and already load-bearing ⇒ no bespoke-no-seam stop. **A `sota-researcher` confirmation is a pre-build gate:** confirm no marker-scanner is a better adopt-target than a native scan, and confirm the conventional marker set worth recognising (see the first open question).

## Open Questions

- **Which markers?** `TODO` and `FIXME` are certain; do we also recognise `XXX`, `HACK`, `BUG`, `NOTE`, `OPTIMIZE`? Case sensitivity and word-boundary rules (avoid matching `TODO` inside an unrelated identifier, or `XXX` used as a redaction placeholder rather than a marker). The recognised set is a convention worth confirming, not guessing.
- **Which tier — conventions or git?** The mapping row and iss-99 place this at the conventions tier (a working-tree file scan). But the issue also says "tracked files", which is a git concept, and the conventions adapters are defined as file-surface-only that "never touch git". Decide whether this is genuinely a conventions-tier working-tree scan, or a git-tier scan of tracked files — the two differ in what they see (untracked files, `.gitignore`d generated code) and in which invariant they must respect.
- **Scan scope and the missing primitive.** `SourceContext` today exposes a non-recursive `ListDir` and a bounded `ReadFile`, but **no recursive walk and no grep primitive**. So the adapter must either (a) add a bounded recursive-walk primitive to `SourceContext`, (b) enumerate tracked files via git (which crosses the "conventions never touches git" line), or (c) scan only a fixed set of top-level directories. Which, and what does that imply for the tier decision above?
- **Which files to scan.** All text files, or source files only? Do we skip vendored / generated / `node_modules` / `.git` / dependency trees, and how are those recognised? How is a binary file detected and excluded so the scan does not read blobs?
- **Per-repo caps.** Beyond the existing per-file and per-directory caps, what bounds a whole-tree scan — a maximum number of files walked, a maximum number of markers reported, a maximum scan time — so a giant monorepo does not dominate the probe?
- **Output shape and framing.** Does the `Evidence` cite each marker as `file:line`, or a count plus a small sample of files? How should the packed brief render the markers — a list, a count, grouped by file? What is enough to be useful without dumping thousands of lines?
- **Dedup.** Collapse repeated identical markers, or many markers in one file, into a representative citation — and if so, on what key?
- **Status and confidence thresholds.** Can markers ever *ground* the section, or is the honest ceiling *partial* (markers are a starting thread, not a framed set of open questions)? Does the count drive the confidence (many markers → higher confidence), and where are the thresholds?
- **Interaction with the native adapter.** When a repository has both an abcd record and source markers, the richest-tier-wins rule discards the marker evidence in favour of the native result. Is that acceptable, or should `evidence/open-questions` merge evidence across tiers here — which would be a change to the reduction model (and thus out of this intent's current scope)?

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-272bfa8a4561 -->
Fidelity review — receipt rcp-272bfa8a4561 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:a8e28c84c4ec99b02445f970343b22fb4c1eefe15a8425a1a9e8fc0ea0be7456
Input attestations: diff:tree at de3ba5fa (spc-12 delivered; main after PR #661)@sha256:4e7430d38ef6b7b6bc533fdf0b8b32a6e08a3e2449d0566027d43316036b8178;

Acceptance rollup: MET 4 · MET_WITH_CONCERNS 1 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: convOpenQuestionsSource is registered at the conventions tier for evidence/open-questions, scans the walked tree for TODO/FIXME/XXX/HACK/BUG and cites file:line (marker); the probe test asserts a record-less fixture reports partial at TierConventions citing retry.go:4 (TODO) and store.go:3 (FIXME)
  evidence: internal/core/lifeboat/sources_conventions.go:793 — "func (convOpenQuestionsSource) Section() Section { return "evidence/open-questions" }"
  evidence: internal/core/lifeboat/sources_conventions.go:844 — "citations = append(citations, fmt.Sprintf("%s:%d (%s)", p, i+1, convMarkerName(m)))"
  evidence: internal/core/lifeboat/sources_conventions_test.go:331 — "func TestConvOpenQuestionsPartialFromWorkMarkers"
- ac-2 — MET: with zero markers the adapter returns blank() carrying the searched list (the marker set) and a human question, and says so when the scan was truncated rather than claiming the tree is marker-free
  evidence: internal/core/lifeboat/sources_conventions.go:852 — "if markers == 0 {"
  evidence: internal/core/lifeboat/sources_conventions.go:872 — "question := "What did this project know was unfinished? Its source carries no work markers.""
  evidence: internal/core/lifeboat/sources_conventions_test.go:386 — "func TestConvOpenQuestionsBlankWithoutMarkers"
  evidence: internal/core/lifeboat/sources_conventions_test.go:586 — "func TestConvOpenQuestionsBlankAdmitsAPartialScan"
- ac-3 — MET: the whole probe (marker adapter included — the fixture carries a TODO and a FIXME) is proved to leave every file byte-identical by hashing before and after, and the walk is proved unable to escape the containment root
  evidence: internal/core/lifeboat/probe_test.go:324 — "func TestProbeLeavesEveryFileByteIdentical"
  evidence: internal/core/lifeboat/probe_test.go:329 — ""src/a.go": "package a\n\n// TODO: handle the retry case\n","
  evidence: internal/core/lifeboat/probe_test.go:117 — "func TestWalkFilesCannotEscapeTheContainmentRoot"
- ac-4 — MET: the scan runs under the walk's file, directory-entry and depth caps plus its own citation cap and byte budget, each pinned by a test that proves the walk or scan stops and reports the truncation
  evidence: internal/core/lifeboat/probe.go:51 — "const maxWalkFiles = maxDirEntries"
  evidence: internal/core/lifeboat/probe.go:60 — "const maxWalkDepth = 32"
  evidence: internal/core/lifeboat/sources_conventions.go:753 — "const maxMarkerCitations = 200"
  evidence: internal/core/lifeboat/sources_conventions.go:763 — "const maxMarkerScanBytes = maxPlanTotalBytes // 512 MiB"
  evidence: internal/core/lifeboat/probe_test.go:192 — "func TestWalkFilesStopsAtTheFileCap"
  evidence: internal/core/lifeboat/sources_conventions_test.go:545 — "func TestConvOpenQuestionsStopsAtTheScanBudget"
- ac-5 — MET_WITH_CONCERNS: the reduction is unchanged and deterministic — highest status wins, richer tier on a tie — and the report prints the winning tier per section, so a repo with both a record and markers resolves to the native result (grounded beats the marker adapter's partial ceiling); the concern is that no test exercises both tiers on this section, and the report names the winning tier but does not say marker evidence was discarded
  evidence: internal/core/lifeboat/probe.go:635 — "richer tier wins (a grounded-at-conventions beats grounded-at-git)."
  evidence: internal/core/lifeboat/coverage.go:119 — "fmt.Fprintf(&b, " (%s, %s)", sanitize(string(s.Tier)), sanitize(string(s.Confidence)))"
  evidence: internal/core/lifeboat/sources_conventions_test.go:357 — "func TestConvOpenQuestionsCeilingIsPartial"

Gap audit:
- honoured:
  - a conventions-tier adapter grounds evidence/open-questions on in-code markers, citing files
    evidence: internal/core/lifeboat/sources_conventions.go:30 — "convOpenQuestionsSource{},"
    evidence: internal/core/lifeboat/sources_conventions_test.go:331 — "TestConvOpenQuestionsPartialFromWorkMarkers"
  - honest three-valued status: markers -> partial with evidence, none -> blank with searched and question
    evidence: internal/core/lifeboat/sources_conventions.go:852 — "if markers == 0 {"
  - bounded, safe scanning inside the probe's caps and containment root
    evidence: internal/core/lifeboat/probe_test.go:117 — "TestWalkFilesCannotEscapeTheContainmentRoot"
    evidence: internal/core/lifeboat/sources_conventions_test.go:545 — "TestConvOpenQuestionsStopsAtTheScanBudget"
  - no fabricated evidence: redaction placeholders and prose mentions are not markers
    evidence: internal/core/lifeboat/sources_conventions_test.go:416 — "func TestConvOpenQuestionsIgnoresRedactionPlaceholders"
    evidence: internal/core/lifeboat/sources_conventions_test.go:465 — "func TestConvMarkerRePinsTheRecognisedSpellings"
- diverged:
  - markers may ground or partially ground the section — delivered with a hard partial ceiling; volume moves confidence only
    evidence: internal/core/lifeboat/sources_conventions_test.go:357 — "TestConvOpenQuestionsCeilingIsPartial"
    evidence: internal/core/lifeboat/sources_conventions.go:771 — "const convMarkerMediumConfidence = 10"
- missing: (none)