---
schema_version: 1
id: "iss-2608311351290623"
slug: "the-matching-fold-drops-format-runes-only-so-an-invisible-ma"
severity: "major"
category: "bug"
source: "impl-review"
found_during: "delta review of itd-185 final fixes"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reading/ingest_regime.go"
resolution: "foldForMatching drops Cf, Other_Default_Ignorable_Code_Point and Variation_Selector and applies NFKC, and every blankness rule — the pattern and each declared body field — is judged on the folded text."
impact: fix
resolved_by:
  intent: "itd-185"
  spec: "spc-63"
---

The matching fold drops format runes only, so an invisible MARK still evades the supply-regime gate and, worse, defeats the named-provenance rule unconditionally. The combining grapheme joiner and the variation selectors are general category Mn rather than Cf, and both are designed to render as nothing: a registrative body carrying one inside a keyword renders identically to the registered phrasing and lands with no refusal and no flag, which is the evasion the previous fix states it closed, one category over. The provenance consequence is the serious half. The blankness test trims ASCII whitespace on the raw field, so a pattern consisting of a single zero-width rune is ACCEPTED at all four regimes and the record it writes asserts a provenance it does not carry. That contradicts the criterion's own words, which say an empty or absent pattern refuses the item without exception at any regime, and it contradicts the refusal text in the code beside it. The fix is to widen the drop set to the two adjacent property tables the standard library already ships and to run the blankness test over the folded copy rather than the raw field.

## Grounds

- pursued: an invisible or compatibility-equivalent rune can no longer decide whether a check fires, and a twelve-case innocent-prose battery still lands; an evasion traded for a false refusal would show this wrong
