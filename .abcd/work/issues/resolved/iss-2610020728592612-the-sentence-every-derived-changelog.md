---
schema_version: 1
id: "iss-2610020728592612"
slug: "the-sentence-every-derived-changelog"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/release/ingest.go"
remedy: "Reword sectionNotice so it says a breaking change is stated as an Added line and nothing else about earlier behaviour is claimed until the composer can see the previous release; the shipped sections keep their text, since the changelog is derived and its history is not rewritten, and a release test pins the new wording."
resolution: "sectionNotice now says a breaking change is an Added line that states the break and nothing else about earlier behaviour is claimed, so a breaking cut no longer opens with a sentence its Breaking lines contradict; TestIngestNoticeSaysWhereABreakIsStated pins it. Sections already cut keep their text."
impact: fix
resolved_by:
  commit: "b432dbd88"
---

The sentence every derived changelog section carries (sectionNotice, internal/core/release/ingest.go) says changes to earlier behaviour are not claimed until the composer can see the previous release, yet writableSections files a breaking record, by ruling, as an Added line that states the break, so every breaking cut prints the notice directly above Breaking lines that claim exactly such changes (v0.12.0's section, dc-4 of its docs-currency review, folded into iss-2609302306281487).

## Grounds

- pursued: the next cut's notice agrees with the Breaking lines under it; a derived section whose notice denies a break its own Added lines state would show it wrong
