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
resolution: "one switch: ahoy install --outbound turns on both outbound checks at once, recording outbound.hook and outbound.forge together, so the committed commit-msg hook and the shell guard's forge-text check are one opt-in. --outbound-hook and --outbound-forge are removed: either is refused at exit 2 with a message naming --outbound, and nothing is written. A repository that recorded only one half under them raises outbound.half_on, and the next ahoy install completes it under the settings approval, recording the other key and, where only the forge-text half was on, writing the hook; completing it never writes abcd.binary"
impact: breaking
resolved_by:
  commit: "06333892658848ab003a8323ac143e8d07d32052"
---

ahoy install has two opt-ins for the one outbound policy, --outbound-hook for the commit-message check and --outbound-forge for the shell guard's forge-text check, so a repository can be left with one half on and a person has to know to pass both. The product thinker ruled on 2026-10-15 that a single opt-in turns on both the commit-message check and the forge-text check.
