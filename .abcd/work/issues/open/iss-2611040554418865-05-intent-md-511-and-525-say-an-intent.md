---
schema_version: 1
id: "iss-2611040554418865"
slug: "05-intent-md-511-and-525-say-an-intent"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/superseded/itd-20-top-level-abcd-dispatcher.md"
remedy: "Set bundle: null and bundle_at_supersession on itd-20."
---

05-intent.md:511 and :525 say an intent in superseded/ with kind_at_supersession: bundle-member carries bundle: null and bundle_at_supersession: <bundle-id>. intents/superseded/itd-20-top-level-abcd-dispatcher.md has kind_at_supersession: bundle-member, keeps bundle: spc-83-operator-surfaces and has no bundle_at_supersession.
