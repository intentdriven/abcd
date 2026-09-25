---
schema_version: 1
id: "iss-238"
slug: "brief-05-internals-01-agents-md-still-reserves-abcd-audit-fo"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "intent-planning-prep"
found_at: ".abcd/development/brief/05-internals/01-agents.md"
resolution: "already fixed at tip: the conformance surface became lint (spc-29, 4cdcf051; brief 16-lint.md), so no shipped audit surface contradicts the /abcd:audit reservation"
impact: internal
resolved_by:
  commit: "4cdcf05165cd34dd898e0dff8a5d26f5b328c7b5"
---

brief/05-internals/01-agents.md still reserves /abcd:audit for 'compliance / hash-chain' (itd-16's umbrella), contradicting the shipped repo-conformance audit surface documented in 04-surfaces/16-audit.md (itd-85 lineage).

## Grounds

- pursued: the agents chapter no longer reserves /abcd:audit against a shipped audit surface; shown wrong if a shipped audit verb and a contradicting reservation reappear in the brief
