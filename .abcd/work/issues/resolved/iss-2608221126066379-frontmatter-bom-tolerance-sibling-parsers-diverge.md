---
schema_version: 1
id: "iss-2608221126066379"
slug: "frontmatter-bom-tolerance-sibling-parsers-diverge"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "bughunt round 7 merge-gate dual review"
found_at: "internal/core/intent/intent.go"
resolution: "frontmatter.Close is the reader's own block walk (BOM trimmed at line 0 only, delimiters by IsDelimiter); intent's three writers, the changelog body, the record page body and peers' title route through it. record-lint's title and agent scope ask frontmatterOpen, and the disposition parser, the launch prose gate and the site stripper trim the BOM at line 0. Each sibling has a BOM-led test watched fail first."
impact: fix
resolved_by:
  commit: "7d8d1b71ca70d93f676bc8aac94aabecf2b3dcf3"
---

frontmatter.Fields trims a leading UTF-8 BOM (iss-2608220134344680) but the sibling parsers that promise byte-exact parity with it do not: intent's setFrontmatterFields (internal/core/intent/intent.go:129, whose comment claims it matches frontmatter.Fields's delimiter tolerance exactly) refuses a BOM-led record the reader accepts, so intent mutations on such a record fail-closed with 'no leading frontmatter block'; changelog's bodyStart (internal/core/changelog/source.go:60, 'so the two never disagree') returns 0 and leaks the whole frontmatter block into the derived changelog body; lint's recordTitle (internal/core/lint/schema.go:554) hand-rolls the comment skip with neither TrimBOM nor the multi-line-comment state. Sweep the siblings onto the shared tolerance or narrow the parity comments to the truth.

## Grounds

- pursued: we expect one exported walk plus line-0 BOM trims to make every frontmatter reader agree with frontmatter.Fields on a BOM-led record; it is shown wrong if any reader of a record's bytes still refuses, mis-titles or leaks the block of a BOM-led record Fields reads
