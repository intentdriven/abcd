---
schema_version: 1
id: "iss-18"
slug: "phase-order-violation-phase-0-scopes-itd"
severity: "major"
impact: internal
category: "inconsistency"
source: "agent-finding"
found_during: "intent-dependency-sweep"
found_at: ".abcd/development/roadmap/phases/phase-0-foundations.md"
resolution: "Edge downgraded, not rescheduled: itd-37's own design says the capture + enforcement half (MG001-MG004, the Phase 0 discipline registration) ships independently and only the extraction-to-memory trigger waits on itd-36 (the partial-ship fallback is designed behaviour). blocked_by: [itd-36] replaced with builds_on: [itd-1, itd-36] and the rationale made explicit in the memory-routing paragraph, so Phase 0 scoping itd-37 no longer violates phase order."
---

phase-order violation: Phase 0 scopes itd-37 whose declared blocker itd-36 is Phase 2; split the hard-on-half edge (capture half vs memory-routing half) or reschedule