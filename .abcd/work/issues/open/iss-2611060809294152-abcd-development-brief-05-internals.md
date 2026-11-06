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
---

.abcd/development/brief/05-internals/03-configuration.md:59 says twenty-two keys plus a `meta` block. Lines 8 and 236 of the same chapter say twenty-three, and the JSON sample at lines 94-198 holds twenty-three keys outside `meta`.
