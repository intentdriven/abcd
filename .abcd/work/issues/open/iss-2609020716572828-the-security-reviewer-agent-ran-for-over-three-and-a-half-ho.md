---
schema_version: 1
id: "iss-2609020716572828"
slug: "the-security-reviewer-agent-ran-for-over-three-and-a-half-ho"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "agents/security-reviewer.md"
---

The security-reviewer agent ran for over three and a half hours on the ahoy branch with no report, while a fresh copy of the same review given an explicit budget of about twenty-five tool calls reported in eight minutes and found the same class. Review prompts should carry a budget and a report-what-you-have instruction, and the orchestrator should time-box a review rather than wait; the run's BRIEF for lane agents already says validation runs in the foreground because a backgrounded gate never wakes the agent that started it, and the same harness shape applies here.
