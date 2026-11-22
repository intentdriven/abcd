---
schema_version: 1
id: "iss-2611060809294152"
slug: "abcd-development-brief-05-internals"
severity: "nitpick"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md"
remedy: "Change the count at 03-configuration.md:59 to twenty-three."
resolution: "The count was already brought into line (dddab8bef made it twenty-three, 2854547b5 twenty-four at every site, matching the 24 keys outside meta in the sample), so the literal remedy would now reintroduce the mismatch; this commit adds TestConfigurationChapterKeyCountMatchesSample, which holds every count site to the sample's key count and fails on the chapter as the issue found it."
impact: internal
resolved_by:
  commit: "266574bd54ea073ee8fb13421a10cd5995f9a11d"
---

.abcd/development/brief/05-internals/03-configuration.md:59 says twenty-two keys plus a `meta` block. Lines 8 and 236 of the same chapter say twenty-three, and the JSON sample at lines 94-198 holds twenty-three keys outside `meta`.

## Grounds

- pursued: the chapter's three count sites agree with the number of keys outside meta in its JSON sample; the new test failing on the current tree, or a count site disagreeing with the sample, would show it wrong
