---
schema_version: 1
id: "iss-2609290300462829"
slug: "the-implement-loop-s-lane-receipt-refusal-returns-the-strict"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/receipt.go"
---

The implement loop's lane receipt refusal returns the strict decoder's message raw, the class iss-2609290218032954 fixed in scribe, release, ideate and lifeboat. The receipt is written by the implementer agent, so it is a host payload, and internal/core/implement/loop/receipt.go:179 formats jsonstrict.Decode's error with %v: an undeclared field or a repeated key is named by the receipt's own spelling, and a token or a home path in that key reaches the terminal and the transcript. receipt_test.go:193 pins the raw key (unknown field "verdict"), so the fix updates that pin. Fix direction: pass the message through scanner.RedactRefusal with the repository root, failing closed to a description. The run state (state.go:274) is out of the class: the loop is its only writer. Detector: a receipt carrying an undeclared key with a home path in it is refused without the path in the error, still naming the key.
