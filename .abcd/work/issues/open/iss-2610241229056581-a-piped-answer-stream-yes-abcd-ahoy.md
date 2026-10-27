---
schema_version: 1
id: "iss-2610241229056581"
slug: "a-piped-answer-stream-yes-abcd-ahoy"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "setup redesign build, flow lane, 2026-10-24"
origin: researcher-authored
production_mode: hand-written
found_at: "ahoy install's piped answer path (branch feat/setup-redesign-flow)"
remedy: "route a piped stream through the same plan screen, answering approve.plan with its first line, and keep the person-only questions out of a piped stream as today"
---

a piped answer stream (yes | abcd ahoy install) is never shown the new plan screen; it still meets the old one-approval-per-category order, so the scripted path and the terminal path ask different questions.
