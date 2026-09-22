---
schema_version: 1
id: "iss-2609221820487644"
slug: "the-attribution-gate-should-let-a-dependabot-dependency-bump-through-without-a-human-re-authoring-it"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "PR 655 blocked on 2026-09-22; the bump landed by hand as PR 659"
origin: researcher-authored
production_mode: hand-written
found_at: "scripts/check-attribution.sh (check_ident); AGENTS.md, Attribution and acknowledgements"
---

The attribution gate should let a dependabot dependency bump through without a human re-authoring it. The product thinker asked for this on 2026-09-22 after PR 655 sat blocked with auto-merge armed and every other check green. Two things must be said precisely, because the request names the trailer and the trailer is not what fails. The gate refuses a bot on IDENTITY: check_ident reads dependabot[bot] as a machine in the author role on two independent signals, the forge's [bot] name suffix and the bot mailbox, and it deliberately stays silent about the missing Assisted-by trailer so the remedy is not misread, its own comment saying that adding a trailer is not the fix for a dependency bump and landing it as a human is. No review clears an identity refusal, which is why an armed auto-merge looks like it is waiting for a reviewer when it is waiting for something no reviewer can give. The rule is deliberate and stated in AGENTS.md: the contributor graph is built from the author and committer fields, a machine there asserts an authorship it does not hold, and a squash merge re-appends a mis-identified branch author as a co-author, so the consequence, that a dependabot pull request is not mergeable as authored, is written down as intended. Wanted: a way for a dependency bump to land without a person re-authoring it every time, without letting a machine into the contributor graph on any other change. Candidate shapes for the decision to weigh: the gate exempts a commit whose diff touches only go.mod and go.sum on a branch the forge marks as dependabot's, with the exemption named in the refusal it would otherwise raise; or the repository takes dependabot's bumps through a workflow that re-authors them as the human owner before the gate runs, so nothing about the gate changes; or dependabot is turned off for this repository and bumps are made by hand on a schedule. This reverses a stated rule, so it is an ADR plus a brief invariant before any code moves, not a quiet edit to the script.
