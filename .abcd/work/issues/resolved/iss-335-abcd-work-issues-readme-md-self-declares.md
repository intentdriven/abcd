---
schema_version: 1
id: "iss-335"
slug: "abcd-work-issues-readme-md-self-declares"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "bughunt-round-2"
found_at: ".abcd/work/issues/README.md"
resolution: "issues/README.md store contract now lists impact (required in resolved/) and agent-observation source, and states the required --impact flag on resolve"
impact: internal
---

.abcd/work/issues/README.md self-declares the store contract with strict validation but omits the impact field (mandatory in resolved/, gated by the issue_impact_valid blocker) and the agent-observation source value (accepted by capture and used by live records) — a reader hand-authoring a resolved record per the contract fails preflight

## Evidence

- `.abcd/work/issues/README.md:9-10,29` -- self-declares the store contract, strict validation; Required (:33-45) and Optional (:47-61) omit impact; source enum (:42-44) omits agent-observation.
- capture/capture.go:65-69 validSources includes agent-observation; capture/workflow.go:170-178 requires impact on resolve; record-lint.json:28 issue_impact_valid blocker gates resolved/; brief 06-capture.md documents both.

## Refuter verdict -- CONFIRMED (substantive, lower end)

154/154 resolved records carry impact; 0/145 open; the one wontfix none -- exactly the folder-conditional semantics the README never states. 10+ live records use source agent-observation. Residue of iss-57 (which updated code+brief, not this README). Nothing lints .abcd/work/ (outside record-lint roots). Fix: add impact (required/valid in resolved/, enum) to the Optional list beside resolution; add agent-observation to the source enum; state the required --impact flag on resolve in the capture-verb section.
