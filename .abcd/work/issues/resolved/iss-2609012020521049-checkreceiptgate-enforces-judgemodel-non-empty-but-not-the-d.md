---
schema_version: 1
id: "iss-2609012020521049"
slug: "checkreceiptgate-enforces-judgemodel-non-empty-but-not-the-d"
severity: "major"
category: "process"
source: "user-observation"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/lint.go"
resolution: "checkReceiptGate refuses a judgeModel that is a floating alias: floatingJudgeModel (internal/core/lint/lint.go) rejects an id naming latest or carrying no version or date digit, and TestReceiptGateRefusesFloatingJudgeModel proves opus, claude-opus, claude-opus-latest and claude-opus-4-latest are refused while claude-opus-4-8, Claude Fable 5, claude-opus-5 and a dated id pass, so every committed receipt stays valid. The runbook states the shape the gate checks."
impact: fix
---

checkReceiptGate enforces judgeModel non-empty but not the documented pinned-snapshot rule. The release-gate runbook, receipt.example.json and the iss-35 plan all state that a receipt's judgeModel is a pinned model snapshot, never a floating alias, yet checkReceiptGate (internal/core/lint/lint.go) refuses only a blank judgeModel: a receipt carrying a bare family name such as opus or an alias such as claude-opus-latest passes the gate, so the auditability the runbook promises is carried by the receipt author's honesty alone. The v0.7.0 receipts record this gap in their _reviewProvenance. The fix must refuse a judgeModel that carries no version or date component or that names a floating alias, with a test, while leaving every committed receipt (all of which carry a family plus version id) valid.

## Grounds

- pursued: the documented rule is expressible by shape without a vendor catalogue, so the gate enforces the floor the runbook already states; whether pinned should mean a dated snapshot rather than a versioned family id is a stricter reading no record has decided and this leaves open
