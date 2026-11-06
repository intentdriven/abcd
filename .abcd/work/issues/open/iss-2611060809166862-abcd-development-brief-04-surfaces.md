---
schema_version: 1
id: "iss-2611060809166862"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md"
remedy: "Say in 05-intent.md that the reason is optional except for a supersession, or require `--reason` on every reclassify."
---

.abcd/development/brief/04-surfaces/05-intent.md:297-298 says every change appends a date, from-kind, to-kind and reason to `reclassification_history`, and line 89 gives the shape as `{ date, from, to, reason }`. `--reason` is required only with `--kind superseded` (internal/core/intent/reclassify.go:194-198), so `abcd intent reclassify <itd-N> --kind bundle-member --bundle <name>` exits 0 and writes an entry with no `reason` key.
