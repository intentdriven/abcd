---
schema_version: 1
id: "iss-2609262011046013"
slug: "the-consistency-review-request-carries-no-findings-json"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/consistency.go"
resolution: "The consistency request carries a Findings shape section rendered from the payload struct the consistency ingest decodes, through the renderer the fidelity request uses, with both ends of a finding shown."
impact: fix
resolved_by:
  commit: "e0d6dcad054be076025937c219be4a8a89004a9a"
---

The consistency review request carries no findings-JSON shape and names no place to read one. consistencyPromptBody in internal/core/intent/consistency.go states the five classes and the rubric, while the findings shape (_type, receipt_id, verifier, policy, and findings carrying class, severity, summary, explanation and two ends of path and quote) is published only in the Role 2 section of agents/intent-auditor.md, so a reviewer working from the request alone learns the shape from ingest refusals. It is the Role 2 sibling of iss-2609181121305984 on the fidelity review request, confirmed while fixing that record; the same remedy applies: the request states the shape rendered from the struct the ingest decodes, never a second copy.

## Grounds

- pursued: a reviewer working from the consistency request alone writes findings the ingest decodes; a stated shape that fails the ingest's strict decode would show it wrong
