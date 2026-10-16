---
schema_version: 1
id: "iss-181"
slug: "brief-lint-contract-python-era-catalogue"
severity: "major"
category: "documentation"
source: "user-observation"
found_during: "2026-08-03 maintainer disposition on iss-37"
found_at: ".abcd/development/brief/05-internals/06-lint.md"
resolution: "06-lint.md drops the Python-era rule catalogue for a present-tense pointer at internal/core/lint and the lint configs"
impact: internal
resolved_by:
  commit: "b760a67c"
---

06-lint.md section 1 (the lint-code namespace) is a catalogue of a predecessor project's Python-era implementation — the `IL001`–`RC007` code families and the class names they cite (`IntentLinter`, `BundleLinter`, `PromptLinter`, `SpecLinter`) describe code this repo does not carry. Re-filed here from iss-37 (2026-08-03 maintainer disposition), which is re-scoped to its three doc-claim instances.

The removal is the **maintainer's decision**, explicit and not open for reinterpretation by the round that implements it:

- The catalogue is **removed, never archived**. It documents another project's implementation, not this one, so it has no historical claim on the brief.

The replacement form is the **orchestrating session's recommendation**, and it stands unless the maintainer overrides it in a later trusted comment before this issue is picked up:

- The catalogue is **not replaced with a Go-equivalent table**. The Go lint engine has no numbered-code scheme, and minting one is taxonomy design rather than doc repair; a hand-typed catalogue would already be stale against this cycle's own rules (`record_schema`, `index_drift`, `delivery_state`, `context_citation_currency`).
- Section 1 is **reworked in generic language**: what the lint engine covers, stated in categories and in the present tense, pointing at the live sources of truth — the armed rules in `.abcd/record-lint.json` and `.abcd/docs-lint.json`, and the engine in `internal/core/lint`. Any literal enumeration of rules must be generated and gated (an `index_drift`-style marked region), never hand-kept — the same discipline iss-38 and iss-42 established.

The **gate cross-check detector** moves here from iss-37: every named gate, lint code, Makefile target, or workflow step in the record resolves to a live definition, and a planned check is written as an intent rather than in the present tense. Its scoping is unsettled: both questions below MUST be settled in this body before any round implements the detector.

1. **Scan root** — which trees the check reads (`.abcd/development/` alone, or the repo root markdown and `docs/` as well), and whether dated plans, ratified ADRs and superseded intents are in scope or exempt as chronological records.
2. **What "a live definition" means for a prose-named gate** — a Makefile target and a workflow step name resolve mechanically, but a gate named only in prose has no symbol to resolve against, so the check needs either a declared registry of gate names or a rule that confines it to citations carrying a resolvable handle.

From the 2026-08-05 maintainer disposition on iss-43, a **candidate extension** of the detector's scope, to be settled alongside those two questions rather than committed to here: README capability and status claims, where a claimed-shipped capability resolves to a wired verb and a phase claim resolves against the roadmap (dropped from iss-43, whose one surviving instance is doc repair).

Related fact recorded in the iss-40 decision [dcn-2608020000000008](../../decisions/dcn-2608020000000008-iss-40-one-canonical-glossary-chosen-as.md): 06-lint.md's `TM003`–`TM011` rows and the `H1` schema-load row claim "Delivered" schema validation that no Go code implements. That is the same phantom-claim class and is in scope for this issue.
