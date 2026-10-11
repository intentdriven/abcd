---
schema_version: 1
id: "iss-2608261041020419"
slug: "context-surface-coverage-understates-cli"
severity: "nitpick"
category: "documentation"
source: "agent-finding"
found_during: "bughunt-a/round-7"
found_at: ".abcd/work/CONTEXT.md"
resolution: "CONTEXT.md: named the sub-verb CLI-tree pass and the surfaces (agent, hook, operator-internal verbs, Direction A) that stay uncovered."
impact: internal
resolved_by:
  commit: "2811321ba4b6adcd06bf05d14101c4bd76d26cdc"
---

CONTEXT.md's surface_coverage sharp-edge says it reads only the command and skill surfaces; its sub-verb pass also reads the CLI command tree. The bullet says the agent, hook, and CLI-verb surfaces are not covered, but subverbs.go reads the committed surface.json command-tree snapshot and files findings under surface_coverage for surface-bearing verbs, including a reverse sweep. The staleness errs in the safe direction (understating enforcement) so it is low-risk, but CLI-verb surfaces are now partly covered; genuinely uncovered are the operator-internal verbs, bare CLI-only verbs, and all of Direction A. Fix: reword to name the sub-verb pass and the truly-uncovered set.