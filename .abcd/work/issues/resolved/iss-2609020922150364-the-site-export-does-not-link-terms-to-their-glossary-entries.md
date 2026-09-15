---
schema_version: 1
id: "iss-2609020922150364"
slug: "the-site-export-does-not-link-terms-to-their-glossary-entries"
severity: "minor"
category: "future-work-seed"
source: "agent-finding"
found_during: "glossary-fix-2026-09-02"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/site"
resolution: "The glossary is exported as its own page set — /record/glossary/, one page per entry, in the explorer's navigation — and the first use of each entry on a rendered record or entry page is a link to it, aliases included, case-insensitive and whole-word. A use inside a code span, a heading, an existing link, or on the term's own page is left as the record wrote it, and dedup is by entry rather than by spelling so a term's aliases together earn one link. internal/core/site reads the entries through glossary.ScanInRoot rather than parsing them a second time; a repository with no glossary gets no pages, no navigation entry and no links."
impact: additive
resolved_by:
  commit: "746a5d2c"
---

The glossary now names every sense of the record's colliding terms (iss-2609012245352480), but the site export reads no glossary: internal/core/site carries no reference to it, so a rendered page shows 'phase', 'plan' or 'surface' as plain words and a reader on the site cannot reach the entry that disambiguates them. The record's ask was that the site links terms to the glossary; that half is Go work and stays open here. Directions, none adopted: the site renderer wraps the first occurrence of each glossary term on a page in a link to its entry; or every rendered page carries a glossary sidebar listing the terms it uses; or the glossary is exported as its own page set and chapters link by hand.

## Grounds

- pursued: we expect a reader who meets 'phase', 'plan' or 'surface' on a rendered page to reach the entry that fixes which sense is meant without leaving the site; it is wrong if the links are so dense the prose is unreadable, if one enters a code span, heading or existing link, or if a repository keeping no glossary is refused a build.
