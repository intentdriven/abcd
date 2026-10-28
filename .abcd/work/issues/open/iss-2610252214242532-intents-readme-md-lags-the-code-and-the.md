---
schema_version: 1
id: "iss-2610252214242532"
slug: "intents-readme-md-lags-the-code-and-the"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "brief single-source study, 2026-10-25, reader S02"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/README.md"
remedy: "Replace the README's kinds, lifecycle, bundle and template copies with pointers to brief § 1 and § 3 and commands/intent.md."
---

intents/README.md lags the code and the record. :38 and :277 list the spc-83-operator-surfaces members as planned (itd-24, 63 and 69 shipped; itd-20 superseded). :100 and :132 mark bundle and reclassify as design targets, though bundle.go and reclassify.go ship. :159 says an intent is 'implemented by exactly one spec', against invariant 17 (02-constraints/03-invariants.md:171). Its own :33 names the brief canonical.
