---
schema_version: 1
id: "iss-2610152024327989"
slug: "ahoy-install-has-two-opt-ins-for-the-one"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "product thinker ruling on the outbound opt-ins, 2026-10-15"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/cli.go"
remedy: "Replace both flags with one ahoy install opt-in that records outbound.hook and outbound.forge together, refuse the retired spellings with a message naming the replacement, and have the next ahoy install complete a repository that recorded only one half; prove each with a test watched failing first."
---

ahoy install has two opt-ins for the one outbound policy, --outbound-hook for the commit-message check and --outbound-forge for the shell guard's forge-text check, so a repository can be left with one half on and a person has to know to pass both. The product thinker's answer, 2026-10-15 ~20:15 BST: "One switch for both": a single opt-in turns on both the commit-message check and the forge-text check.
