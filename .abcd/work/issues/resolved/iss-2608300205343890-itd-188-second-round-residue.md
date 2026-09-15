---
schema_version: 1
id: "iss-2608300205343890"
slug: "itd-188-second-round-residue"
severity: "minor"
category: "inconsistency"
source: "impl-review"
found_during: "itd-188 second-round adversarial reviews, 2026-08-30"
found_at: "internal/core/lint/scribecontract_test.go, agents/scribe/fixtures/injection-canary.json, .abcd/development/specs/open/spc-66"
resolution: "spc-66 no longer claims the record_schema validation path exists today and no longer names the guard root as .abcd/work/: all four passages now say the schema arrives with spc-58 and give the root as .abcd/work/issues/. The guard replaces its prose exemption with an explicit idiom allow-list (and/or, read/write, either/or, I/O) and treats every other separator-bearing token as a path, so a bare internal/core, docs/ or internal/README is reported; folding gains HTML-entity and percent decoding, NFKC (golang.org/x/text is already a direct dependency), the five separators NFKC leaves alone, and horizontal whitespace around a separator; any Cf code point is refused outright rather than folded; the comment names exactly the folding done and states that version pairs and URLs are flagged as paths by design. The canary's control becomes a field-value shape only an obeyed run emits, since verbatim transcription puts the planted word in a faithful output, and each input item carries the identifier the dispositions and the outstanding list refer to."
impact: internal
---

itd-188 second-round review residue: spc-66 still asserts the record_schema validation path retracted elsewhere and names the guard root as .abcd/work/ while its amended table says .abcd/work/issues/; the guard's prose exemption (fewer than two separators and no dot) admits a bare shipped-tree directory such as internal/core or a spaced-separator path; the canary's must_not_contain token is one the verbatim-transcription rule requires the output to carry, so no faithful run can satisfy it; the extractor does not fold U+2571, U+29F8, U+FF3C, HTML-entity or percent-encoded separators, does not refuse format (Cf) code points, and its comment overclaims every spelling; legitimate prose such as version pairs and URLs is flagged as outside paths without a comment saying so; the canary's input items carry no identifier while the expected output refers to item-1 and item-2.
