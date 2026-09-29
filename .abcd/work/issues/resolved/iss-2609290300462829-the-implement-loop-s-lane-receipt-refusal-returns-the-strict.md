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
resolution: "Fixed: every site classified. receipt.go 179 (decoder message) redacted through scanner.RedactRefusal; 113/116 (run, lane, branch) and 201 (a malformed commit name) described; 235 (a path outside the lane's directory) described; 240-246 (a path inside it) named redacted, and a read error reported by its cause alone. Kept: 95 (the operand), 110/135 (integers), 168 (an abcd-built path), 204-216 (a validated 40-hex name, shortened). receipt_test.go:193's pin (unknown field \"verdict\") holds unchanged, since redaction keeps a plain key. state.go 274 stays out of the class: the loop is its only writer."
impact: fix
resolved_by:
  commit: "06432db4c"
---

The implement loop's lane receipt refusal returns the strict decoder's message raw, the class iss-2609290218032954 fixed in scribe, release, ideate and lifeboat. The receipt is written by the implementer agent, so it is a host payload, and internal/core/implement/loop/receipt.go:179 formats jsonstrict.Decode's error with %v: an undeclared field or a repeated key is named by the receipt's own spelling, and a token or a home path in that key reaches the terminal and the transcript. receipt_test.go:193 pins the raw key (unknown field "verdict"), so the fix updates that pin. Fix direction: pass the message through scanner.RedactRefusal with the repository root, failing closed to a description. The run state (state.go:274) is out of the class: the loop is its only writer. Detector: a receipt carrying an undeclared key with a home path in it is refused without the path in the error, still naming the key.

## Grounds

- pursued: a receipt carrying a home path in an undeclared key, the run, lane, branch, a commit name or the report path is refused without the path and still names what is missing; a refusal quoting the receipt would show it wrong
