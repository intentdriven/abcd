---
schema_version: 1
id: "iss-366"
slug: "termsafe-sanitize-masks-four-zero-width"
severity: "nitpick"
category: "security"
source: "agent-finding"
found_during: "bughunt-round-3"
found_at: "internal/termsafe/termsafe.go"
resolution: "Extended termsafe isZeroWidth to mask U+2060 WORD JOINER, the U+2061-2064 invisible operators, and U+00AD SOFT HYPHEN; test invariants updated (watched fail then pass)."
impact: fix
---

termsafe.Sanitize masks four zero-width runes (U+200B/200C/200D/FEFF) but not U+2060 WORD JOINER (the modern successor to the deprecated FEFF-as-joiner it does mask), nor U+2061-2064/U+00AD/Hangul fillers, so those default-ignorable characters pass through and two byte strings render identically
## Evidence

- `internal/termsafe/termsafe.go:104-110` — `isZeroWidth` switch covers only `0x200B,0x200C,0x200D,0xFEFF`.
- `:88-100` — `isBidiControl` does not cover `U+2060`; `:24-38` `Sanitize` passes it through `strings.Map` unchanged (not C0, not DEL, not `0x80-0x9f`).
- Package doc (`:2-6,:18-20`) promises to neutralise "zero-width characters, which … hide text so the rendered line differs from the bytes" — an unbounded promise the enumeration under-delivers.
- Callers carry hostile foreign-repo content (lifeboat, history show, ideate, skew release_tag).

## Adversarial verdict

CONFIRMED (nitpick). Mechanical gap real; severity nitpick — visual-hiding only, no escape/reordering, below the project's "minor" calibration for the escape-injection render gaps (iss-259/iss-264). Notable that the code masks the deprecated FEFF joiner but not its Unicode-designated successor U+2060. Fix: extend the switch to `0x2060,0x2061,0x2062,0x2063,0x2064,0x00AD`; add test invariants.
