---
schema_version: 1
id: "iss-2608311949421873"
slug: "the-include-table-match-grammar-disagrees-with-itself-on-cas"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
---

The include table match grammar disagrees with itself on case for no stated reason: an extension entry is compared with strings.EqualFold while an exact basename entry is compared with ==, so .MD matches but makefile does not match Makefile, and whoever adds a fourth match form has no rule to follow
