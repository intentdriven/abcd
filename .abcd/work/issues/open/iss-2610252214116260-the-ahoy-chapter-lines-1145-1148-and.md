---
schema_version: 1
id: "iss-2610252214116260"
slug: "the-ahoy-chapter-lines-1145-1148-and"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "brief single-source study, 2026-10-25, reader S01"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/01-ahoy.md"
remedy: "State the full early-return condition or point at commands/ahoy.md:483-492, keeping only the not-a-version-stamp invariant in the chapter."
---

The ahoy chapter (lines 1145-1148, and Acceptance 1228-1230) says zero actionable gaps means 'already up to date' with no write, but Install also proceeds for an askable optional offer or a flag that would change a setting, and returns partial for an unverified owned copy (internal/core/ahoy/apply.go:183-229; commands/ahoy.md:489-492).
