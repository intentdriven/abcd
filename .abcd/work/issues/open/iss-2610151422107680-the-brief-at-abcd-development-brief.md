---
schema_version: 1
id: "iss-2610151422107680"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "v0.14.2 release review, iss35 brief-surface crosscheck, content commit 87513ff33"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md"
remedy: "Rewrite .abcd/development/brief/02-constraints/04-naming.md:54-59 to match the shipped surface: The shipped verb `embargo` (bin/abcd-darwin-arm64 --help lists it under Release; commands/embargo.md exists) appears in neither the maritime table, the named exemptions nor the plain exemption list."
---

The brief at .abcd/development/brief/02-constraints/04-naming.md:54-59, .abcd/development/brief/02-constraints/04-naming.md:49-54 says: The remaining surfaces are exempt from the maritime metaphor and 'are registered here so the exemption is on the record'; together with the maritime table and the named exemptions this is the full naming register of the shipped verbs. The shipped surface differs: The shipped verb `embargo` (bin/abcd-darwin-arm64 --help lists it under Release; commands/embargo.md exists) appears in neither the maritime table, the named exemptions nor the plain exemption list. (v0.14.2 release crosscheck, undocumented-surface, raised by opus+sonnet in a32.)
