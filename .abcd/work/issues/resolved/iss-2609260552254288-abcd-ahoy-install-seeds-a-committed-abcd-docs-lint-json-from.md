---
schema_version: 1
id: "iss-2609260552254288"
slug: "abcd-ahoy-install-seeds-a-committed-abcd-docs-lint-json-from"
severity: "minor"
category: "inconsistency"
source: "drift-detection"
found_during: "v0.11.0 release gate: brief-surface cross-check (autonomous run A, abcd-a2)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/apply.go"
resolution: "the prepare-this-repo boundary now says the config ahoy install seeds is the repository's own and committed, matching the installer and the recorded decisions"
impact: fix
resolved_by:
  commit: "d0c18992"
---

`abcd ahoy install` seeds a committed .abcd/docs-lint.json from abcd's own writing-guide rules into a target repository when it has none (internal/core/ahoy/apply.go, defaults/docs-lint.json), while the prepare-this-repo chapter (15-prepare-this-repo.md:125-127) says lint-config JSON the tool supplies is applied and never committed: the installer and the criterion disagree and one of them has to give. Found by the v0.11.0 brief-surface cross-check (x-044).

## Grounds

- pursued: the brief and the command page agree with what ahoy install commits; shown wrong if the next release-gate cross-check reports the installer and the prepare boundary disagreeing again
