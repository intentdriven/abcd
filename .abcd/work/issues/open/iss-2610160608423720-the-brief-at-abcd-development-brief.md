---
schema_version: 1
id: "iss-2610160608423720"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.0 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md"
remedy: "Add readme.parts to the key list and the example at 03-configuration.md:22-34, and count the README question."
---

The brief at .abcd/development/brief/05-internals/03-configuration.md:22 says 'Thirteen keys plus a meta block' and that install asks four questions and writes those values. readme.parts is also read and written in .abcd/config.json (internal/core/ahoy/readme_apply.go; ahoy install --readme-parts, documented in 04-surfaces/01-ahoy.md), and install now asks the README question; the key list, the example JSON and the four-questions count leave it out.
