---
schema_version: 1
id: "iss-2610111600065621"
slug: "unbumped-prompt-edit-passes-agent-diff"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "private security advisory GHSA-5c9h-7q8p-gpj5, filed 2026-10-10"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/agentcontract.go"
remedy: "Replace the +prompt_version: prefix test in promptVersionChanged with a comparison of the parsed frontmatter version on the two sides of the armed range (the merge-base blob for A...B, the worktree for a single revision; a missing old blob counts as an introduction), reporting the unbumped finding when the path changed and the versions are equal; prove both reproduction variants block, a real bump with a changelog entry lints clean, and TestAgentContractUnbumpedChangeOverDiff still passes"
resolution: "The armed agent-diff check now compares the parsed frontmatter prompt_version on the two sides of the range (the merge-base for A...B, A for A..B, the working tree for a single revision), read with ls-tree and cat-file, instead of looking for an added diff line; a prompt absent from the old side counts as an introduction. Proved by TestAgentContractUnbumpedVersionLineIsNoBumpOverWorktree, TestAgentContractUnbumpedVersionLineIsNoBumpOverRange and TestAgentContractThreeDotRangeComparesTheMergeBase, with TestAgentContractRealBumpAndIntroductionLintClean, TestAgentContractUnbumpedChangeOverDiff and TestAgentContractDiffRunsNoRepoDiffDriver still passing (internal/core/lint)."
impact: fix
---

record-lint's armed agent-diff check counts any added diff line starting `+prompt_version:` as a version bump, so an instruction edit under `agents/` passes at an unchanged `prompt_version` when the diff also adds a body line `prompt_version: <same>` or adds one trailing space to the frontmatter line.

Private security advisory GHSA-5c9h-7q8p-gpj5 (draft, severity medium). Present since v0.6.8. GHSA-q333-3p8r-xc2f's pins stop a diff driver forging the line, not git's own diff carrying it. Full text, evidence and reproduction: .abcd/.work.local/scratch/security-drain-2026-10-11/advisories/ in the main checkout's local tier. This record stays uncommitted until its fix lands; the fix commit adds it directly to resolved/.

Evidence (main 3d171421a): `promptVersionChanged` (internal/core/lint/agentcontract.go) returns true on the first diff line with prefix `+prompt_version:` and never compares the parsed version on the two sides. Confirmed by reading the code at that commit; not re-run.
