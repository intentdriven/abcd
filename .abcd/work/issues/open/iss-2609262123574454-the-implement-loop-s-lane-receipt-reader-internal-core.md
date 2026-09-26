---
schema_version: 1
id: "iss-2609262123574454"
slug: "the-implement-loop-s-lane-receipt-reader-internal-core"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/receipt.go"
---

The implement loop's lane receipt reader (internal/core/implement/loop/receipt.go readReceipt) decodes the receipt strictly by hand, refusing an unknown field and a second document, but reads a repeated key last-wins: a receipt carrying "exit_code":1,"exit_code":0 (or a case twin such as EXIT_CODE) verifies as a passing definition of done. Every other trust-boundary JSON reader refuses a repeated key through internal/core/jsonstrict; this one was written before jsonstrict reached its base, so its strict decode is an inline copy that lacks the check.
