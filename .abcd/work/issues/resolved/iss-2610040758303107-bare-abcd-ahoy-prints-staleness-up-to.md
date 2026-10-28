---
schema_version: 1
id: "iss-2610040758303107"
slug: "bare-abcd-ahoy-prints-staleness-up-to"
severity: "minor"
category: "ux"
source: "managed-repo"
found_during: "downstream brief-authoring lab report, 2026-10-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/vintage.go"
remedy: "Say what the comparison was made against (the installed plugin's pin or the checkout tip) and name abcd update --check as the way to compare with the newest release; test: the bare ahoy text for a fresh vintage names its reference and the update-check verb."
resolution: "The fresh staleness verdict that bare abcd ahoy and abcd --version print now names its reference: 'up to date with the checkout tip (<rev>)' for a source build, and 'up to date with the plugin manifest pin (<tag>); run `abcd update --check` to compare with the newest release' for a pinned install. commands/ahoy.md tells the host to relay the line whole."
impact: fix
resolved_by:
  commit: "cfe70bb92ea93e0895bd5be0b30d48f63428cc4e"
---

Bare abcd ahoy prints 'staleness: up to date' when the running binary matches its own install's pin, a comparison made from disk alone by design (itd-111; internal/core/ahoy/vintage.go, Staleness), and the line says neither what it compared against nor that abcd update --check is the verb that asks for the newest release. A downstream user on v0.9.0, several releases behind, read 'up to date' as current and kept working against bundled guidance that predated the retirement of phases and milestones.

## Grounds

- pursued: a user on a pinned install whose binary matches its pin now sees which pin was matched and the verb that asks for the newest release; TestFreshStalenessNamesItsReference would show it wrong by failing if the fresh line drops the reference, the tag or the update-check verb
