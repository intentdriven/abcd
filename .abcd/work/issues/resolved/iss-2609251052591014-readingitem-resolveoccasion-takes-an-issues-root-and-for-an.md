---
schema_version: 1
id: "iss-2609251052591014"
slug: "readingitem-resolveoccasion-takes-an-issues-root-and-for-an"
severity: "minor"
category: "tech-debt"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/readingitem/readingitem.go"
resolution: "ResolveOccasion takes the repository root and resolves an intent occasion by reading the intent store alone (recordid.LookupOne)."
impact: internal
resolved_by:
  commit: "3606c9e9ff33a893b95fa6b39b7b8355a7e0bba2"
---

readingitem.ResolveOccasion takes an issues root and, for an itd-N occasion, recovers the repository root by suffix-stripping .abcd/work/issues off it, while its one caller (intent/condition.go) joins that suffix on only for it to be stripped; the lookup then builds a full recordid.NewResolver over every record family (intents, specs, issues, ADRs) to find one intent, so an unreadable spec store refuses an intent occasion. The leaf should take the repository root and look up the intent store alone.

## Grounds

- pursued: an intent occasion resolves from the repository root with no suffix arithmetic and with an unrelated family's store unreadable; TestResolveOccasionReadsOnlyTheIntentStore and TestLookupOneReadsOnlyTheIdsFamily would fail if either returned
