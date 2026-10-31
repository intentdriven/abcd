---
schema_version: 1
id: "iss-2610312044511316"
slug: "in-facilitator-mode-auto-the-landing-s"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "landing PR #1301 (run-2610310228179894, lane-1)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/land.go"
remedy: "When the lane's last validation round passed and the records commit changes only record files over the validated head, have the landing mint the review receipt for the records commit from that round's returns, saying so in the run record; keep the hand path for any other head."
---

In facilitator.mode auto the landing's merge gate waits for a review receipt naming the records commit, but the loop never mints one from its own validation: lane-1 of run-2610310228179894 passed round 4 (ruthless-reviewer SHIP, security-reviewer APPROVE at 45a7ce2c5) and the landing still held PR #1301 unarmed at its records commit 28611e181 until a fresh reviewer pair was run and recorded by hand with abcd land --record-review.
