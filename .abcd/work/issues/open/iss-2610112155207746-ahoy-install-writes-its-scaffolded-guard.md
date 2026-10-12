---
schema_version: 1
id: "iss-2610112155207746"
slug: "ahoy-install-writes-its-scaffolded-guard"
severity: "minor"
category: "ux"
source: "impl-review"
found_during: "ruthless review, run-2610112109503709, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/banlist_scaffold.go"
remedy: "Waits on a product-thinker ruling: should abcd ever refresh a scaffolded hook already committed in an adopter's repository (a version marker bump that ahoy detects as a stale-hook gap and offers to rewrite, with the diff shown), or are adopters' committed hooks theirs once written? Once ruled, implement the chosen route in ahoy's detection and apply, with a test of an installed v1 hook carrying the old header."
---

ahoy install writes its scaffolded guard hooks into an adopted repository only when they are missing (internal/core/ahoy/banlist_scaffold.go, the banlist.hook_missing branch through createContained), and never rewrites an installed v1 hook. So a repository that ran ahoy before a fix to a scaffolded hook keeps the old text: after iss-2610111820465940 and iss-2610112148115897 make the core.hooksPath advice conditional, every earlier adopter's committed .githooks/pre-commit and prepare-commit-msg keep the unconditional header, and ahoy reports no gap for it. Found by the ruthless review of the lane for iss-2610111820465940.
