---
schema_version: 1
id: "iss-2610281503005495"
slug: "on-28-oct-2026-one-login-reached"
severity: "minor"
category: "ux"
source: "agent-observation"
found_during: "autonomous-run-2026-10-28"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim"
remedy: "Count the ceiling per machine (or per copy) as well as per login, show each machine's share in the refusal, and say which config raises which ceiling; carry this into the cross-machine release theme."
refines: [iss-2610231004140527]
---

On 28 Oct 2026 one login reached reservation.max_live (40 of 40) with every hold live and in use: 24 from 15 store worktrees on this machine and 16 from the person's other machine, all counted against the one total. A new pre-cut fix's reserve was refused (exit 2), and the cap was raised to 50 by hand in the machine config. Two machines' autonomous runs under one login share a ceiling that neither machine can see the other's share of.
