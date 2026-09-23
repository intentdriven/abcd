---
id: itd-96
shipped_in: v0.4.0
slug: disembark-reads-a-repo-s-naming-and-internals-conventions-in
spec_id: spc-13
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: additive
---

# Disembark Reads A Repo's Naming And Internals Conventions Into Its Lifeboat

## Press Release

> **A lifeboat now grounds a project's naming rules and its internal architecture even when the project never kept an abcd record.** When `abcd disembark` probes a repository, two new conventions-tier adapters read the places a team naturally documents these things — a `GLOSSARY.md` or naming table for `constraints/naming`, and an `ARCHITECTURE.md`, a `docs/architecture/` tree, and the package layout for `internals`. A repository with conventional docs but no `.abcd/` record now packs a lifeboat whose naming and internals sections are grounded in the repo's own files, instead of coming back blank because only a hand-authored abcd brief could fill them.
>
> "The service I inherited had a clear ARCHITECTURE.md and a glossary, but no abcd record at all," said Bob, who was rebuilding it. "The old lifeboat left 'internals' and 'naming' empty and told me to write them myself — with the answers sitting right there in the repo. Now disembark reads what the team already wrote down and carries it into the lifeboat."

## Why This Matters

Disembark grounds each brief section through tiered `Source` adapters (`internal/core/lifeboat/`), with the richest present tier winning and anything nothing can ground returned as a first-class blank (adr-35). Two extractable sections are grounded **only** at the native tier today:

- **`constraints/naming`** is grounded solely by `nativeNamingSource`, which reads the brief glossary under `.abcd/development/brief/glossary/`. There is no conventions-tier (or git-tier) adapter, so on a repository without a record the section falls through to blank.
- **`internals`** is grounded solely by the generic `nativeBriefSource` reading `.abcd/development/brief/05-internals/`. There is no conventions-tier or git-tier adapter for it either, so it too blanks on any record-less repository.

Both are `KindExtractable` sections (adr-36): a blank here is **coverage debt abcd can close with a better adapter**, not a human-owned question — precisely the gap the M2 cross-repo gate probe surfaced. The mapping contract (`internal/core/lifeboat/mapping.go`) already predicts a *partial* conventions status for both rows and names what they should read ("glossary, naming registry, reserved-vocabulary tables" for naming; "package layout, architecture docs, the internals chapters" for internals) — a prediction no adapter yet delivers.

**Stale-premise correction (verified against the code).** The originating issue also named `glossary`, but a glossary conventions adapter **already exists**: `convGlossarySource` in `sources_conventions.go` grounds the `glossary` section from `GLOSSARY.md` or a `docs/glossary*` page. So `glossary` is already covered and is explicitly **out of scope** here — this intent's real scope is the two genuinely missing conventions adapters: `constraints/naming` and `internals`.

Naming rules and internal structure are exactly what a rescuer needs early: the reserved vocabulary that must not be renamed, and the shape of the system they are about to change. When a project wrote these down conventionally but never adopted abcd, the lifeboat should still carry them.

Promoted from the open ledger issue iss-100.

## What's In Scope

- **A conventions-tier `Source` adapter for `constraints/naming`.** It grounds the section from a repository's naming documentation — a glossary or a naming/reserved-vocabulary table (the exact source and its relationship to the existing glossary adapter is an open question below). Same `Source` interface, read-only `SourceContext` surface, never writes.
- **A conventions-tier `Source` adapter for `internals`.** It grounds the section from a repository's architecture documentation (`ARCHITECTURE.md`, a `docs/architecture/` or `docs/explanation/` tree) and/or its package layout (top-level source directories). Same interface and read-only contract.
- **Honest three-valued status for both.** Real documentation → grounded; a thin signal (e.g. package dirs but no architecture prose) → partial; nothing → a blank carrying `Searched` and the human `Question`, per the blank contract.
- **Bounded, safe reads.** Both adapters stay inside the probe's existing safety envelope (containment root, `maxProbeReadBytes`, `maxDirEntries`) and add whatever bounds a layout scan needs.
- **Two coverage-report rows that now ground where they used to blank** on a conventional, record-less repository — the visible readout of the closed gaps.

## What's Out of Scope

- **A `glossary` conventions adapter — already shipped.** `convGlossarySource` grounds `glossary` today; this intent must not add a second adapter for that section. (Stale-premise correction to iss-100.)
- **Changing the tier-reduction model** or the mapping contract's section list — this intent adds two adapters to the existing contract.
- **Git-tier archaeology of naming or architecture** across history — the new adapters read the current working tree only.
- **Deep semantic architecture extraction** (call graphs, dependency graphs, generated diagrams) — the internals adapter reads what the team wrote down and the surface package layout, not a synthesised model of the code.
- **New extraction dependencies** — both adapters reuse abcd's own read surface (see SOTA).
- **The missing `evidence/open-questions` conventions adapter** — that is iss-99 / itd-95, a sibling intent.

## Scope Conditions

None stated.

## Acceptance Criteria

> _BDD format, per the [itd-1 discipline](../disciplines/itd-1-acceptance-gates.md). Some bars reference decisions still open in § Open Questions; they are written to hold whichever way those resolve._

- **Given** a git repository with no `.abcd/` record but with naming documentation (a glossary or a reserved-vocabulary table), **when** `abcd disembark probe` runs, **then** `constraints/naming` is reported non-blank and cites the file(s) it read.
- **Given** a repository with no record but with an `ARCHITECTURE.md` (or a `docs/architecture/` tree) and a recognisable package layout, **when** the probe runs, **then** `internals` is reported non-blank and cites the architecture doc and/or the layout it read.
- **Given** a repository with neither naming nor architecture documentation and no recognisable layout, **when** the probe runs, **then** `constraints/naming` and `internals` each come back blank with a populated `Searched` list and a human `Question` — an honest blank, not a fabricated section.
- **Given** any source repository, **when** either new adapter's `Probe` runs, **then** the source tree is byte-for-byte identical afterwards — both are read-only by construction.
- **Given** a repository whose only glossary signal is a `GLOSSARY.md`, **when** the probe runs, **then** the `glossary` section is still grounded by the pre-existing `convGlossarySource` and no duplicate or conflicting `glossary` adapter has been introduced — this intent adds naming and internals only.
- **Given** a repository with both an abcd record and conventional naming/architecture docs, **when** the probe runs, **then** each section resolves to a single deterministic result via the richest-tier-wins rule, and the coverage report explains which tier grounded it.

## Prior Art

- **`internal/core/lifeboat/sources_conventions.go`** — the conventions-tier adapter pattern the two new adapters follow; **`convGlossarySource`** in particular is the already-shipped sibling that grounds `glossary` (and the reason `glossary` is out of scope here).
- **`internal/core/lifeboat/sources_native.go` — `nativeNamingSource`** (grounds `constraints/naming` from the brief glossary) and the generic **`nativeBriefSource`** for `internals` (reads `brief/05-internals/`) — the only adapters grounding these sections today, both native-tier. The new adapters ground the same sections one tier down.
- **`internal/core/lifeboat/mapping.go`** — the contract rows for `constraints/naming` and `internals`, each predicting a *partial* conventions status and naming what to read; this intent delivers the adapters that make the predictions true.
- **[adr-35](../../decisions/adrs/0035-lifeboat-as-coverage-experiment.md)** (lifeboat as a coverage experiment) and **adr-36** (extractable vs human-owned sections) — both sections are `KindExtractable`, so their blanks are coverage debt to close. Promoted from the open ledger issue iss-100 (found during the M2 cross-repo gate probe).
- **[one-canonical-primitive](../../principles/one-canonical-primitive.md)** — the new adapters reuse the existing conventions-adapter helpers (README/prose thresholds, directory checks) rather than inventing parallel scanning logic.

## SOTA

> _Per the [sota-per-intent principle](../../principles/sota-per-intent.md): existing alternatives + rough maturity, then a chosen path. First-pass sketch for a draft; the confirmation is a pre-build gate._

- **Naming / vocabulary extraction.** There is no production-mature tool that extracts a project's reserved vocabulary from its docs; the closest are documentation generators and glossary linters, which do a different job. The native floor (read a glossary or naming table) is the realistic option.
- **Internal-architecture extraction.** Alternatives exist for *structure* — dependency/architecture visualisers (`dependency-cruiser`, `structurizr`, `godoc`/package docs, architecture-linters) — *usable-to-mature*, but they synthesise a model or render diagrams; none grounds an abcd brief section or emits the coverage contract, and each is an external dependency. Reading what the team already wrote (`ARCHITECTURE.md`, `docs/architecture/`) plus the surface package layout is the honest, dependency-free floor.
- **Chosen path — Path 2 (basic-with-seam)** for both adapters. Native reads over the existing read-only `SourceContext` surface, returning `Evidence` like every other adapter. No new dependency. The seam is the `Source` interface: a richer external extractor could later back either section without changing the reduction model.
- **Verdict — Path 2.** No new dependency ⇒ no gate-1 stop; the seam is the existing `Source` interface, already load-bearing ⇒ no bespoke-no-seam stop. **A `sota-researcher` confirmation is a pre-build gate:** confirm no architecture-extraction tool is a better adopt-target than a native read, and confirm the conventional file/paths worth recognising (see open questions).

## Open Questions

- **Naming vs glossary — what is naming's distinct source?** `convGlossarySource` already reads `GLOSSARY.md` / `docs/glossary*` for the `glossary` section, and the *native* naming adapter derives naming from the same brief glossary. So does the conventions naming adapter read the **same** glossary doc (one file grounding both `glossary` and `constraints/naming`), or does it look for a **distinct** naming registry / reserved-vocabulary table (and what are that file's conventional spellings — `NAMING.md`? there is no strong standard)? If naming simply re-reads the glossary, what, if anything, distinguishes the two sections' conventions grounding?
- **Which paths map to `internals`?** Candidates: `ARCHITECTURE.md` (and spellings), a `docs/architecture/` or `docs/design/` tree, `docs/explanation/` (Diátaxis), and the package layout. Which are authoritative, and in what preference order for the citation?
- **What counts as "package layout", and the missing primitive.** Top-level source directories (`internal/`, `pkg/`, `src/`, `lib/`, `cmd/`, `app/`, …) are readable with the existing non-recursive `ListDir`. But a deeper layout scan needs a recursive walk primitive that `SourceContext` does not expose (the same gap itd-95 raises). Is a top-level directory listing enough for `internals`, or is a bounded recursive walk needed — and is that a shared primitive both intents should land?
- **Extraction heuristics and status thresholds.** When is `internals` *grounded* vs *partial*? An `ARCHITECTURE.md` with real prose (mirroring the README `convProseBytes` threshold) → grounded, but a bare package listing with no architecture doc → partial? For `constraints/naming`, does a stub glossary reach only partial (as the native adapter already decides via its body-bytes threshold)?
- **Confidence levels.** What drives high vs medium vs low confidence for each adapter — presence of an authored doc vs a layout-only signal?
- **Reserved-vocabulary spellings.** If naming looks for a dedicated table rather than the glossary, which filenames and heading shapes count (a `## Reserved words` section, a `NAMING.md`, a naming-conventions doc under `docs/`)?

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-b6cd3962cc41 -->
Fidelity review — receipt rcp-b6cd3962cc41 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:70a5ee6f6a60d717ff864c0fcff1733a7af53630b3139074282db4146cbd4545
Input attestations: diff:tree at de3ba5fa (spc-13 delivered; main after PR #661)@sha256:4e7430d38ef6b7b6bc533fdf0b8b32a6e08a3e2449d0566027d43316036b8178;

Acceptance rollup: MET 5 · MET_WITH_CONCERNS 1 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: convNamingSource grounds constraints/naming from a naming document (or docs/naming*) as partial, citing the file, with a lower-confidence glossary fallback; tests assert the section is partial at TierConventions citing NAMING.md and a docs/naming page
  evidence: internal/core/lifeboat/sources_conventions.go:508 — "func (convNamingSource) Section() Section { return "constraints/naming" }"
  evidence: internal/core/lifeboat/sources_conventions.go:512 — "p := ctx.FindFirst(convNamingDocNames...)"
  evidence: internal/core/lifeboat/sources_conventions_test.go:638 — "func TestConvNamingPartialFromNamingDoc"
  evidence: internal/core/lifeboat/sources_conventions_test.go:665 — "func TestConvNamingPartialFromDocsNamingPage"
- ac-2 — MET: convInternalsSource reads ARCHITECTURE.md (or an architecture/design/explanation tree) and the package layout under internal/pkg/src/lib/cmd/app, citing both; the test asserts internals is non-blank citing the doc and the layout entries
  evidence: internal/core/lifeboat/sources_conventions.go:611 — "func (convInternalsSource) Section() Section { return "internals" }"
  evidence: internal/core/lifeboat/sources_conventions.go:556 — "var convLayoutRoots = []string{"internal", "pkg", "src", "lib", "cmd", "app"}"
  evidence: internal/core/lifeboat/sources_conventions_test.go:765 — "func TestConvInternalsCitesArchitectureAndLayout"
- ac-3 — MET: with no naming doc, no glossary, no architecture doc and no recognised layout both adapters return blank() with a populated searched list and a human question; a bare fixture is tested for both sections
  evidence: internal/core/lifeboat/sources_conventions.go:536 — ""What names and reserved vocabulary are fixed? No naming document and no glossary found.","
  evidence: internal/core/lifeboat/sources_conventions.go:653 — "question := "How is this system built internally? No architecture document and no recognisable package layout.""
  evidence: internal/core/lifeboat/sources_conventions_test.go:1012 — "func TestConvNamingAndInternalsBlankWithoutSignals"
- ac-4 — MET: the byte-identical probe test's fixture carries NAMING.md, ARCHITECTURE.md and an internal/ layout, so both new adapters run inside the hashed-before-and-after proof
  evidence: internal/core/lifeboat/probe_test.go:324 — "func TestProbeLeavesEveryFileByteIdentical"
  evidence: internal/core/lifeboat/probe_test.go:332 — ""NAMING.md": "# Naming\n\nA voyage is never called a run.\n","
  evidence: internal/core/lifeboat/probe_test.go:333 — ""ARCHITECTURE.md": "# Architecture\n\nA core, and a shell that formats it.\n","
- ac-5 — MET: the conventions registry carries exactly one glossary adapter (convGlossarySource) beside the two new ones, and a glossary-only fixture is tested to keep glossary grounded by it while naming falls back to it at lower confidence without displacing it
  evidence: internal/core/lifeboat/sources_conventions.go:27 — "convGlossarySource{},"
  evidence: internal/core/lifeboat/sources_conventions.go:527 — "Sources: []string{g + " (glossary fallback — no dedicated naming document)"},"
  evidence: internal/core/lifeboat/sources_conventions_test.go:696 — "func TestConvNamingFallsBackToGlossaryWithoutDisplacingIt"
  evidence: internal/core/lifeboat/sources_conventions_test.go:1042 — "func TestHasConventionsCoversNamingAndArchitecture"
- ac-6 — MET_WITH_CONCERNS: the unchanged reduction is deterministic (highest status wins, richer tier on a tie) and the report prints the winning tier per section, so a native record's grounded result beats the partial conventions reading; the concern is that no test exercises a record plus conventional docs on these two sections specifically — spc-13 relies on the reduction's existing tests
  evidence: internal/core/lifeboat/probe.go:635 — "richer tier wins (a grounded-at-conventions beats grounded-at-git)."
  evidence: internal/core/lifeboat/coverage.go:119 — "fmt.Fprintf(&b, " (%s, %s)", sanitize(string(s.Tier)), sanitize(string(s.Confidence)))"
  evidence: internal/core/lifeboat/sources_conventions.go:606 — "The ceiling is StatusPartial by construction"

Gap audit:
- honoured:
  - a conventions-tier adapter for constraints/naming
    evidence: internal/core/lifeboat/sources_conventions.go:28 — "convNamingSource{},"
  - a conventions-tier adapter for internals reading architecture docs and package layout
    evidence: internal/core/lifeboat/sources_conventions.go:29 — "convInternalsSource{},"
    evidence: internal/core/lifeboat/sources_conventions_test.go:765 — "TestConvInternalsCitesArchitectureAndLayout"
  - bounded reads: the layout scan cites at most 50 packages and reports a truncated walk
    evidence: internal/core/lifeboat/sources_conventions.go:562 — "const maxLayoutCitations = 50"
    evidence: internal/core/lifeboat/sources_conventions_test.go:891 — "func TestConvInternalsBoundsItsLayoutCitations"
  - no second glossary adapter
    evidence: internal/core/lifeboat/sources_conventions_test.go:696 — "TestConvNamingFallsBackToGlossaryWithoutDisplacingIt"
- diverged:
  - real documentation -> grounded — delivered with a partial ceiling for both sections; documentation quality moves confidence, not status
    evidence: internal/core/lifeboat/sources_conventions.go:606 — "The ceiling is StatusPartial by construction"
    evidence: internal/core/lifeboat/sources_conventions.go:518 — "Status: StatusPartial,"
- missing: (none)