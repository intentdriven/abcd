---
schema_version: 1
id: "iss-2611060809312249"
slug: "abcd-development-brief-05-internals"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md"
remedy: "Restate the cost as about 8k tokens, and say the rule count includes the entries defined in code."
---

.abcd/development/brief/05-internals/03-configuration.md:1553 says SHELL's injected block costs about 2k tokens for the bundled registry. `*SHELL` through `abcd hook prompt-router --json` in a repository with no `.abcd/guard.json` injects 33,320 bytes, about 8,000 tokens at four bytes a token, in 63 rules. internal/core/guard/defaults/guard.json holds 60 entries; the other three rules name entries defined in code.
