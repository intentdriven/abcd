---
schema_version: 1
id: "iss-2608300232312077"
slug: "itd-188-fourth-round-residue"
severity: "minor"
category: "inconsistency"
source: "impl-review"
found_during: "itd-188 fourth-round ruthless review, 2026-08-30"
found_at: ".abcd/development/specs/open/spc-66, agents/scribe.md, agents/scribe/fixtures/injection-canary.json, internal/core/lint/scribecontract_test.go"
resolution: "spc-66's Tests bullet no longer says the guard folds every separator spelling: it now states the decode-to-fixpoint, the two folds that remain, and the non-ASCII refusal that replaced the lookalike table. The definition's untrusted-input banner says an embedded instruction is named in the refusals and is not a fidelity flag on its own, since a flag needs two pieces of material that disagree; the canary's behaviour text names the same contradiction its exemplar emits, and the exemplar's refusals now carry the embedded instructions the behaviour text says are refused. The residual-entity comment states that a bare ampersand followed by a letter is refused by design, Q&A included, because an entity is exactly that shape. The bypass table asserts the finding CLASS per case, so the path, traversal, non-ASCII, format and residual-encoding branches are each pinned by their own cases rather than by any sibling's finding."
impact: internal
---

itd-188 fourth-round residue: spc-66's Tests section still says the guard folds every separator spelling to a slash, while the guard now folds only the ASCII reverse solidus and spaced separators and refuses every other non-ASCII spelling — a maintainer trusting the spec could allow a lookalike in prose believing it still folds; the definition's untrusted-input banner says an embedded instruction is a thing a fidelity flag reports while the flag section requires two disagreeing pieces of material, and the canary's behaviour text names a different candidate flag than its exemplar emits; the residual-entity regex refuses any ampersand followed by a letter (Q&A) without the comment saying so; the bypass table asserts only that some finding exists, so the Cf branch's message is pinned by one case.
