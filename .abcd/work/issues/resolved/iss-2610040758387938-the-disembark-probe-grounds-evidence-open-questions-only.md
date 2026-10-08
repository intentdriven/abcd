---
schema_version: 1
id: "iss-2610040758387938"
slug: "the-disembark-probe-grounds-evidence-open-questions-only"
severity: "minor"
category: "bug"
source: "managed-repo"
found_during: "downstream brief-authoring lab report, 2026-10-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lifeboat/sources_native.go"
remedy: "Let the two dedicated sources also ground from the section's own brief file as nativeBriefSource does, and give every partial verdict a reason naming what was found and what would ground it; test: a fixture with an authored 03-open-questions.md and no open issues grounds the section, and every partial in a probe's output carries a non-empty reason."
resolution: "The disembark probe's open-questions and tradeoffs adapters now also ground from the section's own brief file (authored grounds, stub is partial), and every partial verdict at every tier carries a reason naming what was found and what would ground it, carried into coverage as `reason` and rendered as `why partial:`; partials are built only through one constructor that requires the reason."
impact: fix
resolved_by:
  commit: "86fdecff59aad4500817b6899637d6514c669f5c"
---

The disembark probe grounds evidence/open-questions only from open issues and intents, and evidence/tradeoffs only from ADR alternatives and the decision log (internal/core/lifeboat/sources_native.go, nativeOpenQuestionsSource and nativeTradeoffsSource). Neither reads the repository's own brief file for its section (03-evidence/03-open-questions.md, 03-evidence/04-tradeoffs.md), which every other brief section's source reads, so a downstream project with both files authored saw them ignored. A partial verdict also carries no reason: Evidence fills Searched and Question only on a blank, so a section rated partial does not say what was found or what would ground it.

## Grounds

- pursued: a repository with authored 03-open-questions.md / 04-tradeoffs.md and no open issues, ADR alternatives or decision log now probes those sections as grounded, citing the file, and no partial row in any probe output lacks a reason; shown wrong if TestNativeEvidenceSectionsGroundFromTheirBriefFiles, TestProbeEveryPartialCarriesAReason, TestEverySourcePartialCarriesAReason or TestPartialIsBuiltOnlyThroughTheConstructor fails, or if `abcd disembark probe` on such a repo still reports those sections blank or a partial without a reason.
