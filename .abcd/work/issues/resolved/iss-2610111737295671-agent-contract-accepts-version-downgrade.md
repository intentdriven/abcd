---
schema_version: 1
id: "iss-2610111737295671"
slug: "agent-contract-accepts-version-downgrade"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "Fable review of security fix lane X, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/agentcontract.go"
remedy: "Require the new side's version to be greater than the old by semver, or its changelog entry to be absent from the old side's changelog; prove with a test that a downgrade to an existing entry is a finding, watched failing first, while a real bump stays clean"
resolution: "The armed agent-contract check now counts only a higher prompt_version, by semver precedence, as a bump; an edit that moves a prompt back to an older version is reported naming both versions, and an equal version is reported unbumped as before. Proved by TestAgentContractVersionDowngradeIsNoBump (0.3.0 back to 0.2.0 with both entries present, reported over HEAD, main...HEAD and main..HEAD) and TestComparePromptVersions, watched failing first, with TestAgentContractRealBumpAndIntroductionLintClean still passing (internal/core/lint)."
impact: fix
resolved_by:
  commit: "580f8db27"
---

The agent-contract bump check counts any different prompt_version as a bump, so an instruction edit that moves an agent's prompt_version back to an older value whose changelog entry already exists passes both halves of the check with no new entry. Found by the Fable review of lane X on 2026-10-11 (probe: main...HEAD and main..HEAD both report zero findings); predates the fix for iss-2610111600065621 and is outside its remedy.
