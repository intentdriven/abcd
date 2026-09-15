---
schema_version: 1
id: "iss-2608300240410495"
slug: "scribe-guard-admits-control-character-splice"
severity: "minor"
category: "security"
source: "impl-review"
found_during: "itd-188 fifth-round security review, 2026-08-30"
found_at: "internal/core/lint/scribecontract_test.go (code-point loop)"
resolution: "The guard's code-point loop refuses unicode.IsControl outside LF and TAB, with its own finding class and message: a control splits a path match exactly as a format code point does, while a viewer renders it invisibly, so the spliced halves read as one allowed path. The carriage return is in the refused set because the definition is LF-only. Three bypass cases pin the branch — CR, VT and NUL, each splicing a trailing traversal, watched red first. The canary's control string becomes the double-quoted token rather than a colon-and-value shape, which is the smaller of the two fixes offered and is spacing-independent, so compact JSON cannot slip past it; the test also asserts the exemplar omits the control it forbids, since a model answer that trips its own control is one no faithful run can satisfy."
impact: internal
---

The scribe guard's code-point loop refuses format code points and every non-ASCII code point but not ASCII control characters, so a carriage return, vertical tab, form feed, NUL, escape or DEL splits a path match the way a format code point would; a trailing traversal on the far side of the control character stands alone, produces no finding, and a viewer that renders the control invisibly shows .abcd/work/issues/.. as allowed material. Refuse unicode.IsControl outside LF and TAB with its own class marker; the test should also assert the canary exemplar omits the control shape.
