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
---

`abcd ahoy install` seeds a committed .abcd/docs-lint.json from abcd's own writing-guide rules into a target repository when it has none (internal/core/ahoy/apply.go, defaults/docs-lint.json), while the prepare-this-repo chapter (15-prepare-this-repo.md:125-127) says lint-config JSON the tool supplies is applied and never committed: the installer and the criterion disagree and one of them has to give. Found by the v0.11.0 brief-surface cross-check (x-044).
