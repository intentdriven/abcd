---
schema_version: 1
id: "iss-2610311024153333"
slug: "identity-block-cannot-carry-a-lockup"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/positioning/block.go"
remedy: "Add optional Name, Descriptor and Purpose bullets to the identity block beside Title, Tagline and Pitch: the parser reads them, a surface may require and template them, the site's brand is the name (else the title's first word), the hero's eyebrow and the page title are the headline (name and descriptor, else the title), and the purpose renders and is checked under the tagline. A block without them reads exactly as before. Prove it with a parser test over the six fields and their fallbacks and a site test that the hero renders the lockup, passes the gates and refuses a drifted purpose. Grounds: internal, the product thinker's 2026-10-31 identity ruling."
resolution: "the identity block reads optional Name, Descriptor, Purpose and Listing fields; surfaces template them, and the site leads its hero and page title with the name and descriptor and checks the purpose"
impact: additive
---

A repository's identity block holds only a title, a tagline and a pitch, so a product cannot be named apart from its title, and it has no field for a descriptor that says what kind of thing the product is and whom it is for, or for a purpose that says why it exists. A repository whose title expands its name cannot lead its surfaces with the name and a descriptor, and the site reads its brand from the title's first word, which breaks once the title is the expansion alone. Found while the product thinker set out a three-line lockup (descriptor, tagline, purpose) beside a title and a technical pitch, 2026-10-31.
