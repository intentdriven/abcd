---
schema_version: 1
id: "iss-2609301000153822"
slug: "two-adr-files-claiming-one-id-are-read"
severity: "minor"
category: "security"
source: "user-observation"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
remedy: "Refuse at both ends: a record-lint blocker adr_id_unique over the ADR store (filename number and frontmatter id, both vintages, case- and padding-insensitive, sharing validateIDUnique with the intent, issue and spec rules), and a resolver lookup that returns an ambiguity error naming every claimant instead of the first. Grounds: the three sibling uniqueness rules already establish the pattern in this codebase; no outside practice is involved."
resolution: "adr_id_unique refuses two ADR files claiming one id (filename number or frontmatter id, both vintages, any spelling), and the resolver's lookup refuses an ambiguous id naming every claimant; abcd adr-N resolves through it."
impact: fix
resolved_by:
  commit: "22dbb24fc"
---

Two ADR files claiming one id are read first-wins: recordid's resolver kept the first path in name order for an id two files claim, and record-lint had unique-id rules for intents, issues and specs only, so an accepted 0037-a.md beside a proposed 0037-x.md settled every reader of adr-37 (abcd adr-37 rendered the twin; a start-blocker check reading the status would settle on it).

## Grounds

- pursued: two files answering to one ADR id fail record-lint and make abcd adr-N refuse naming both; a duplicate that passes record-lint, or a lookup that returns one of two claimants, would show it wrong
