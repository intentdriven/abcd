---
schema_version: 1
id: "iss-2610141705135681"
slug: "the-brief-glossary-names-a-shipped-abcd"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "v0.14.1 release review, iss35 brief-surface crosscheck (Direction B), content commit 74165cf97"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/README.md"
remedy: "Mark the grill sub-verb as a design target, or remove it from the glossary until it ships."
refines: [iss-2609091956001547]
---

The brief glossary names a shipped /abcd:intent grill sub-verb, but abcd intent --help lists no grill sub-verb, and the brief elsewhere calls that promotion a design target only.
