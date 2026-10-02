---
schema_version: 1
id: "iss-2610020730472210"
slug: "a-lane-whose-landing-finds-no-preflight-receipt-naming-its"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/land.go"
remedy: "Per ruling DR6d-2: make the missing receipt a contention like the merge wait, write the time the wait began once on the lane's landing and keep it across steps, show 'waiting for its full check (since HH:MM)' in the step's blocked line, implement status and the status block, and keep the remedy naming how the receipt is minted; who starts the check (DR6d-2b) stays out."
---

A lane whose landing finds no preflight receipt naming its head refuses the whole build step (land.go landPush, a plain refusal), so every other lane stops while one lane waits for its full check, and nothing shows when that wait began; ruling DR6d-2 (the product thinker, 2026-10-02) says the other lanes carry on, the lane waits for its receipt like the merge wait (contend, not refuse), and the status shows 'waiting for its full check (since HH:MM)'.
