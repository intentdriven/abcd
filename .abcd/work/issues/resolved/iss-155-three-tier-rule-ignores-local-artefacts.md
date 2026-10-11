---
schema_version: 1
id: "iss-155"
slug: "three-tier-rule-ignores-local-artefacts"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "managed-repo NEXT.md privacy-leak investigation 2026-07-29"
found_at: "internal/core/audit/rule_layout.go"
resolution: "three-tier-layout flags NEXT.md, scratch/ and logs/ found directly in a committed tier, with a move-to-.work.local fix per finding"
impact: fix
---

three-tier-layout verifies tier presence and the .work.local gitignore but never that local-tier artefacts are absent from committed tiers — a NEXT.md, scratch/, or logs/ under .abcd/work/ or .abcd/development/ passes clean. Field incident: NEXT.md (handover file, local-tier by convention) lived at .abcd/work/NEXT.md in a public repo and carried host infra details; no rule flagged the misplacement.