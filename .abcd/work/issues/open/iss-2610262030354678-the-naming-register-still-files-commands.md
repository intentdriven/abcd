---
schema_version: 1
id: "iss-2610262030354678"
slug: "the-naming-register-still-files-commands"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 18 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md:66"
remedy: "In 04-naming.md replace 'later phases' (line 66) and each 'a later phase' on the dredge, loot, audit and failure_mode_tag rows with 'staged' or a pointer to the draft intent; sequencing is dependencies plus the lifecycle shelves (adr-2609212115255771)."
finding_ref: cfk-bd01696217220132
---

The naming register still files commands under 'later phases'; the scope page says the phase is retired as a unit of the record

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/04-naming.md:66` — "**Reserved meta-development commands** (later phases; named now to prevent collisions)"
- End B: `.abcd/development/brief/01-product/04-scope.md:30` — "retired the phase and the milestone as units of the record"

The naming register heads its reserved commands 'later phases' and tags dredge, loot and audit as 'a later phase'. The scope page says adr-2609212115255771 retired the phase and the milestone as units of the record. The register uses a retired term as a live scheduling bucket.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 18.
