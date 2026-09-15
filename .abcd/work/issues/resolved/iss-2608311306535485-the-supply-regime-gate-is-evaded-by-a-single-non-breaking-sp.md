---
schema_version: 1
id: "iss-2608311306535485"
slug: "the-supply-regime-gate-is-evaded-by-a-single-non-breaking-sp"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
resolution: "Signature matching runs over a folded copy: every Unicode space folded to ASCII, every format rune dropped. The stored text is untouched, and the encoding order is unchanged."
impact: fix
resolved_by:
  intent: "itd-185"
  spec: "spc-63"
---

The supply-regime gate is evaded by a single non-breaking space: RE2's whitespace and word-boundary classes are ASCII-only and termsafe.Sanitize does not mask U+00A0, so a registered signature's own phrasing with U+00A0 between two words produces no match and the item lands with no refusal and no flag. This is NOT the residue itd-185 discloses, which covers a fix proposal or disposition phrased OUTSIDE the registry; this is the registry's own phrasing with one invisible byte substituted, so it is an evasion of the gate rather than a limit of it.

## Grounds

- pursued: four invisible-rune evasions each refuse and innocent prose carrying a non-breaking space is still accepted, so an evasion has not been traded for a false refusal; a signature evaded by a normalisable rune would show this wrong
