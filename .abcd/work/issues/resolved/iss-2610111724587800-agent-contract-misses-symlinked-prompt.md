---
schema_version: 1
id: "iss-2610111724587800"
slug: "agent-contract-misses-symlinked-prompt"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "security fix lane X, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/agentcontract.go"
remedy: "Refuse a symlink under agents/ in the agent-contract rule (record-lint blocker naming the path), or resolve each agents/ entry and judge the target's change in the armed range; prove with a test that a symlinked prompt whose target changed at the same prompt_version is a finding, watched failing first"
resolution: "The agent-contract rule now reports a prompt under agents/ that is a symlink, and an agents directory that is a symlink or lies under one, as a finding naming the path, so an edit made through a link can no longer bypass the bump check. Proved by TestAgentContractRefusesASymlinkedPrompt (a symlinked prompt file and a symlinked agents directory, each edited through the link at an unchanged version, reported over main...HEAD, main..HEAD and main), watched failing first; the rest of the agent-contract tests and record-lint armed over this repository's real prompt bumps stay clean (internal/core/lint)."
impact: fix
resolved_by:
  commit: "fc74c7b0d"
---

record-lint's agent-contract bump check never sees an edit made through a symlink: when agents/<name>.md is a symlink to a file elsewhere in the repository, editing the target changes nothing under agents/ in the diff, so an instruction change ships with no prompt_version bump and no finding. Found by lane X while fixing the unbumped-prompt check (iss-2610111600065621) on 2026-10-11; predates that change. Under abcd-fixes-what-it-adds this is abcd's own check not doing what it claims, so it is abcd's to fix.
