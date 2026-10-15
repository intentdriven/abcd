---
id: dcn-2607280000000007
slug: itd-102-surface-comparison-is-normalised
date: 2026-07-28
---
2026-07-28 — itd-102: surface comparison is normalised CONTAINMENT of each required block field (markup, dashes, wrapping, case folded; trailing sentence punctuation trimmed from the needle), not line equality. Equality would flag abcd's own three conforming surfaces, each of which carries the tagline in a different rendering (inside `<p>`, concatenated with the pitch in the manifest, bolded mid-sentence and line-wrapped in AGENTS.md); containment catches all three iss-143 variants while accepting all three current ones. Consequence: `identity render` fires only on drift, and its template render is a proposal to adopt by hand, not a byte-exact reproduction of conforming prose.
