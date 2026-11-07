---
schema_version: 1
id: "iss-2611060809546274"
slug: "the-packing-history-example-at-03-embark"
severity: "nitpick"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/03-embark.md"
remedy: "Change the example's `dest` to an absolute path and say the field holds the resolved destination."
---

The packing-history example at `03-embark.md:329` shows `"dest": "../abcd-lifeboat"`. Packing with a relative destination records the resolved absolute path, so the example shows a value the field does not hold.
