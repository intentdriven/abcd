---
schema_version: 1
id: "iss-2610310331170759"
slug: "the-library-and-configuration-chapters"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.1 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/07-library.md"
remedy: "Restate user-scope memory in 04-surfaces/07-library.md and 05-internals/03-configuration.md as shipped through memory add and the recall."
refines: [iss-2610151422129661]
---

The library and configuration chapters say no verb writes user-scope memory and nothing resolves it; memory add writes the machine-layer memory.json and the prompt recall reads both layers.
