---
schema_version: 1
id: "iss-2610262035440979"
slug: "the-glossary-reserves-bare-surface-for-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 594 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/surface.md:45"
remedy: "In surface.md allow bare 'surface' inside the identity chapter, commands/identity.md and positioning.json, where the registry itself calls the held texts surfaces, or name the sense 'identity surface' and drop the coined 'rendered surface'."
finding_ref: cfk-8b95fce550438343
---

The glossary reserves bare 'surface' for a verb's front door and 'rendered surface' for identity-held text; the identity chapter uses bare 'surface' for identity-held text throughout

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/surface.md:45` — "The verb takes an object and needs no qualifier; the noun without one always means the front door."
- End B: `.abcd/development/brief/04-surfaces/19-identity.md:6` — "tells the product thinker which surfaces have drifted"

surface.md says the noun without a qualifier always means the front door, and names 19-identity.md as the home of the qualified sense 'a rendered surface'. The identity chapter calls the README strapline, plugin manifest and conventions-file opening 'surfaces' with no qualifier ('which surfaces have drifted', 'A surface can also come back unlocatable') and never writes 'rendered surface', so a reader applying the glossary would read those sentences as about command front doors.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 594.
