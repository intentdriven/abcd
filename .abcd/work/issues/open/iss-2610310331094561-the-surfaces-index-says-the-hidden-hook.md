---
schema_version: 1
id: "iss-2610310331094561"
slug: "the-surfaces-index-says-the-hidden-hook"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.1 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/README.md"
remedy: "Count and name all eleven hook entrypoints in the hook row of 04-surfaces/README.md, pointing at 05-internals/03-configuration.md for the dogfood pair."
---

The surfaces index says the hidden hook verb has nine entrypoints; abcd hook --help registers eleven, dogfood-binary and dogfood-build included, which no surfaces chapter names.
