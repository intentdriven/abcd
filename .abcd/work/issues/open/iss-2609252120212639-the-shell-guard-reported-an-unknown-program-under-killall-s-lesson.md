---
schema_version: 1
id: "iss-2609252120212639"
slug: "the-shell-guard-reported-an-unknown-program-under-killall-s-lesson"
severity: "minor"
category: "ux"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/guard.go"
---

The shell guard reported a command whose program name is a command substitution under whichever registry entry fired first, with that entry lesson: killall-by-name telling an agent to stop a build command by its pid, where the way past the recorded over-block is to spell the program name. The over-block is also wider than the DECISIONS line states: any command word whose basename ends in a substitution, with any operand (review4-guard finding 4).
