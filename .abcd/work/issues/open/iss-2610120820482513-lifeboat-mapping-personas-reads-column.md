---
schema_version: 1
id: "iss-2610120820482513"
slug: "lifeboat-mapping-personas-reads-column"
severity: "nitpick"
category: "drift"
source: "impl-review"
found_during: "reviews of the lane for iss-2610102343100369, run-2610112221301757, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lifeboat/mapping.go"
remedy: "Make the personas row's Reads name what the adapter reads (the brief's personas page, brief/01-product/05-personas.md), regenerate the 00-meta.md block with renderMapping, and sweep every other row's Reads against its adapters in sources_native.go, sources_git.go and sources_conventions.go, fixing each that names material no adapter reads; a test that fails when a row's Reads omits a file its native adapter reads, watched fail first."
---

The lifeboat mapping Table's personas row (internal/core/lifeboat/mapping.go, rendered into .abcd/development/brief/00-meta.md) says its Reads are 'personas registry, press-release quote attributions', but the only personas adapter, nativePersonasSource, reads the brief's own personas page (brief/01-product/05-personas.md) through nativeWithBriefFile and nothing else. A reader of the generated table concludes the probe looks for a registry or attributions it never opens. Raised by the ruthless and security reviews of the lane for iss-2610102343100369, which left it outside that remedy.
