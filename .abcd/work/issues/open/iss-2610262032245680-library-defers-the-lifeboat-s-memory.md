---
schema_version: 1
id: "iss-2610262032245680"
slug: "library-defers-the-lifeboat-s-memory"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 264 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/07-library.md:424"
remedy: "In 07-library.md (the disembark bullet near line 424) drop 'declaring an exact payload is deferred to the disembark spec that wires the packer (adr-28)': the packer shipped under adr-35 with a fixed brief-section file set that has no memory section; say the lifeboat carries no memory payload today, or mark one a design target."
finding_ref: cfk-94738d8e275fb65b
---

Library defers the lifeboat's memory payload to the spec that wires the packer; disembark has the packer shipped with a fixed file set and no memory section

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/07-library.md:424` — "declaring an exact payload is deferred to the disembark spec that wires the packer (adr-28)"
- End B: `.abcd/development/brief/04-surfaces/02-disembark.md:129` — "Each path derives from the brief-section-to-lifeboat-path mapping in `internal/core/lifeboat/mapping.go`: a section grounds into its file where the source supports it"

Library says disembark is designed to export curated project memory into the lifeboat and that declaring the exact payload is deferred to the disembark spec that wires the packer. Disembark records the pack as shipped, every path derived from the brief-section mapping, and its output shape holds the brief, graveyard, rescue, retrospectives, ADRs and issues with no memory section. The deferral waits on a wiring that, as disembark writes it, has already happened without carrying memory.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 264.
