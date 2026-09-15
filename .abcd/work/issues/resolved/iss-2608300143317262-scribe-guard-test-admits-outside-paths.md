---
schema_version: 1
id: "iss-2608300143317262"
slug: "scribe-guard-test-admits-outside-paths"
severity: "major"
category: "security"
source: "impl-review"
found_during: "itd-188 adversarial security review, 2026-08-30"
found_at: "internal/core/lint/scribecontract_test.go, agents/scribe/fixtures/injection-canary.json"
resolution: "The access check now reads the whole definition after folding every separator spelling to /, so a second Inputs heading, a bare or fenced path, a fullwidth solidus and a backslash are all seen; the root narrows to .abcd/work/issues/, a traversal segment is refused outright and the prefix is checked on the cleaned path. TestScribeAccessCheckRefusesEveryBypass is the eight-case control. The canary gains the transcript-as-ledger-context and embedded-summary-ask lures, and plants its must_not_contain token in the payload."
impact: internal
---

The scribe's guard test can be satisfied by a definition that names shipped-tree material: the allow-list root constant is .abcd/work/ (broader than the .abcd/work/issues/ the definition, brief and changelog state); only the first Inputs-prefixed heading is scanned; only single-backtick tokens containing an ASCII slash are read as paths (a bare path, a fenced block, or a fullwidth solidus is invisible); and the prefix check performs no cleaning, so a dot-dot traversal under the prefix passes. The canary also omits two refusals the definition claims (transcript-store or .work.local material handed as ledger context; an embedded ask for a summary) and its must_not_contain token appears nowhere in the input.
