---
schema_version: 1
id: "iss-2610091935325886"
slug: "submodule-inner-diff-runs-submodule-diff"
severity: "minor"
category: "security"
source: "user-observation"
found_during: "security-drain-2026-10-09 lane W sweep"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/agentcontract.go"
remedy: "Add -c diff.submodule=short to gitutil.ExecPins so every isolated diff shows a submodule as a pointer change only, proved by a test whose submodule config names an external diff, watched fail first."
resolution: "gitutil.ExecPins forces diff.submodule=short on every isolated command, so a patch diff (the decisions-append and agent-diff reads included) shows a moved submodule as its pointer change and starts no git diff inside the submodule, whose own diff.external or textconv would otherwise run."
impact: fix
---

record-lint's patch diffs (decisions-append, agent-diff) start an inner git diff inside a submodule when the superproject sets diff.submodule=diff, and git passes that child none of the parent's --no-ext-diff/--no-textconv, so the submodule's diff.external or textconv runs and its output lands in the text the check parses. Sibling of iss-2610090821531570, found by the security-drain-2026-10-09 lane W sweep; kept uncommitted until its fix lands.
