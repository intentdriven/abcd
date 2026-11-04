---
schema_version: 1
id: "iss-2611040555055477"
slug: "06-delivery-01-build-sequence-md-92-says"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/01-build-sequence.md"
remedy: "Mark the clause as a design target."
---

06-delivery/01-build-sequence.md:92 says a vendor memory harvest is an opt-in, read-only source over the memory substrate, unmarked, though the chapter marks unbuilt items as design targets (:111, :129, :134). No Go source reads memory.harvest and no verb or flag offers one; 05-internals/03-configuration.md:684 and :1323 list it as staged. The shipping abcd lab harvest is unrelated.
