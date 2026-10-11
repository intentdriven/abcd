---
schema_version: 1
id: "iss-2609020716572828"
slug: "the-security-reviewer-agent-ran-for-over"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "agents/security-reviewer.md"
resolution: "agents/security-reviewer.md 0.3.0 carries a Budget section: the dispatcher's stated budget or about twenty-five tool calls, then a report of what survived refutation with a NEEDS-INPUT verdict naming the unreached ground, and a dispatcher that hears nothing by its own time box reads the review as NEEDS-INPUT."
impact: internal
resolved_by:
  commit: "909b93d49"
---

The security-reviewer agent ran for over three and a half hours on the ahoy branch with no report, while a fresh copy of the same review given an explicit budget of about twenty-five tool calls reported in eight minutes and found the same class. Review prompts should carry a budget and a report-what-you-have instruction, and the orchestrator should time-box a review rather than wait; the run's BRIEF for lane agents already says validation runs in the foreground because a backgrounded gate never wakes the agent that started it, and the same harness shape applies here.

## Grounds

- pursued: a budgeted review reports within its budget instead of running for hours; a dispatched security review that again runs past its budget with no report would show it wrong
