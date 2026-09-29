---
schema_version: 1
id: "iss-344"
slug: "parallel-hunts-capture-the-same-defect-twice-iss-306-hunt-a"
severity: "minor"
category: "architectural-insight"
source: "user-observation"
found_during: "hunt-A round-1 reconciliation"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker: should the capture-time match (itd-2609212137116617) also read the unmerged records abcd peers lists? That would be an intent of its own."
remedy: "Waits on ruling (should the capture-time match read the unmerged records abcd peers lists): if yes, feed the records the peers package enumerates into the itd-2609212137116617 match at capture time, proven by a test that a near-duplicate minted in a sibling worktree fixture is named by the match; if no, add running abcd peers before a capture to the hunt brief template and close this record as covered by peers."
---

Parallel hunts capture the same defect twice: iss-306 (hunt A round 1, landed late) and iss-339 (hunt B round 2) record the identical githubRemoteRe case-sensitivity bug, minted independently on unmerged branches invisible to each other. Collision-proof ids (itd-114) do not solve this class — it is the semantic sibling of the id collision, sharing the root cause that parallel minters cannot see each other's unmerged mints. Candidate rungs: a capture-time similarity check (itd-84's validator rung), hunt briefings that read open PRs' ledgers, or a shared mint registry as a side effect of itd-114's forge-backed option

## Remedy grounds (2026-09-29)

Why: abcd peers already enumerates what sibling worktrees hold, so the match needs a second input, not a new registry. Rejected: a shared mint registry through the forge, which ruling J12 (2026-09-29) parks as decide later for forge-backed numbering.
