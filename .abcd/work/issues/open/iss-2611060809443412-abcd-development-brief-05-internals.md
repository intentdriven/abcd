---
schema_version: 1
id: "iss-2611060809443412"
slug: "abcd-development-brief-05-internals"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/07-memory.md"
remedy: "Rewrite the bracket table and the sentence at line 193 from `bracketTable`, adding the unknown row and the rule shrinking."
---

`.abcd/development/brief/05-internals/07-memory.md:185-189` gives three brackets. `internal/core/rules/brackets.go:79-85` has four: an unknown row applies when no source gives the fill and sends pages as one line each. MODERATE sends the top-ranked pages in full up to 8 KiB and the rest as one line each. DEPLETED's lines come from each page's own summary, not `index.md` (`internal/core/memory/recallpages.go:26-27`, `internal/core/recall/pages.go:2`), which also makes :193's query of `index.md` wrong. MODERATE and DEPLETED also shrink the rules to a heading and first rule, which the chapter never says.
