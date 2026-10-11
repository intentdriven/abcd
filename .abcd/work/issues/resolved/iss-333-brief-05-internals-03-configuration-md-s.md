---
schema_version: 1
id: "iss-333"
slug: "brief-05-internals-03-configuration-md-s"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "bughunt-round-2"
found_at: ".abcd/development/brief/05-internals/03-configuration.md"
resolution: "brief 05-internals/03-configuration.md development tree redrawn flat by artefact type per adr-30 (intents/ five buckets, decisions/adrs, plans/, principles/, specs/ inside development/); no roadmap/intents literal"
impact: internal
---

brief/05-internals/03-configuration.md's Plugin-internal development namespace tree draws the pre-adr-30 layout: roadmap/intents/{drafts,planned,shipped}, research/adr/, research/phase/, and specs/ under .abcd/ — the record's only map of its own namespace routes a reader to four paths that do not exist; the blocker moved-intents-path rule is defeated only by the ASCII art splitting the string

## Evidence

- `.abcd/development/brief/05-internals/03-configuration.md:390-411,419` -- draws roadmap/intents/{drafts,planned,shipped}, research/adr/, research/phase/, .abcd/specs/.
- Reality: top-level intents/ with five buckets (adr-30:33-35; spec.go intentBuckets); decisions/adrs/NNNN-slug.md; research/{notes,prompting}; SpecsRelDir = .abcd/development/specs (spec.go:37).

## Refuter verdict -- CONFIRMED (substantive, not release-blocking)

adr-30:64-68 names the drawn shape as the rejected alternative; the blocker moved-intents-path (pattern roadmap/intents) is defeated only by the ASCII art line-splitting. The chapter self-contradicts (:205 corrected tier table) and contradicts siblings. Not covered by iss-250/192/237/181/298. Refuted sub-claims (do not carry): .abcd/work/notes/ (unbuilt verb output, legitimately absent) and the agents/ (x15) count (iss-241). Fix: targeted redraw of :390-411 from adr-30 + development/README.md and re-site specs/ inside development/; must never contain the literal roadmap/intents; leave design-target entries (memory/, logbook/, rp/, corpus.json) untouched.
