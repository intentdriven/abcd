---
schema_version: 1
id: "iss-2611040555212928"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "drift"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/17-guard.md"
remedy: "Drop the example from the unseen list and name the entry among what the guard blocks."
---

.abcd/development/brief/04-surfaces/17-guard.md:997-998 gives `rm -rf "$DIR"/*` with DIR unset as a target an allow does not see. `abcd guard check` blocks it, exit 1, under the bundled entry rm-unguarded-variable-path (internal/core/guard/defaults/guard.json:126; commands/guard.md:1082), and likewise the `$DIR/*`, `"${DIR}"/*` and `"$DIR"/x` spellings. The chapter never names the entry.
