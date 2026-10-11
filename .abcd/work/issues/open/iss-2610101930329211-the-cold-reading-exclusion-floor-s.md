---
schema_version: 1
id: "iss-2610101930329211"
slug: "the-cold-reading-exclusion-floor-s"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reading/project.go"
remedy: "Compare the rendered texts after slugStripRe and a case fold rather than through site.Slug, so only headings that really render alike match; a test shows '## Open/Questions' is neither redacted nor refused while '## **Open Questions**' still is."
---

The cold-reading exclusion floor's sameRendering (internal/core/reading/project.go) compares heading titles by site.Slug, which is coarser than 'renders the same': the slug collapses every run of non-alphanumerics to a hyphen, so '## Open/Questions' slugs like '## Open Questions' and is treated as the excluded heading. Since the redactor now shares namesExcludedHeading with the verifier, such a heading is silently redacted rather than refused, so a section that merely slugs alike disappears from the bundle without a word.
