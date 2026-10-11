---
schema_version: 1
id: "iss-2610101930333607"
slug: "a-heading-spelled-with-a-cyrillic"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reading/project.go"
remedy: "Fold confusables before matching, or refuse any heading that is an excluded heading's near-match once non-ASCII letters are set aside, so a homoglyph spelling fails closed; a test assembles '## Аudit Notes' with a Cyrillic A and expects it redacted or refused."
---

A heading spelled with a Cyrillic confusable, '## Аudit Notes' with a Cyrillic A (U+0410), is neither redacted nor refused by the cold-reading exclusion floor, on main as on the redaction-case branch: namesExcludedHeading folds case and compares slugs, and both compare code points, so the excluded section travels into the bundle. The floor discloses homoglyphs as residue in a comment beside namesExcludedHeading, but nothing refuses them.
