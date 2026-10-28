---
schema_version: 1
id: "iss-2610230851529426"
slug: "03-configuration-md-s-user-scope"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.2 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md"
remedy: "Add the four folders to the inventory, with the chapter that owns each."
---

03-configuration.md's user-scope inventory of ~/.abcd.noindex/ omits four folders the binary writes: away/, sessions/, dashboard/ and after-update/.
