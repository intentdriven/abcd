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
resolution: "namesExcludedHeading refuses a near-match: a title agreeing with an excluded heading once invisible runes are dropped, compatibility forms normalised and non-ASCII letters set aside, with one ASCII letter agreeing"
impact: fix
resolved_by:
  commit: "9ed6a6454fd3ac83952f28728121be3b787ad43e"
---

A heading spelled with a Cyrillic confusable, '## Аudit Notes' with a Cyrillic A (U+0410), is neither redacted nor refused by the cold-reading exclusion floor, on main as on the redaction-case branch: namesExcludedHeading folds case and compares slugs, and both compare code points, so the excluded section travels into the bundle. The floor discloses homoglyphs as residue in a comment beside namesExcludedHeading, but nothing refuses them.
