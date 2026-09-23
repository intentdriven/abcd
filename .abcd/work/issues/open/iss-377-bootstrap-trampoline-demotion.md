---
schema_version: 1
id: "iss-377"
slug: "bootstrap-trampoline-demotion"
severity: "minor"
category: "future-work-seed"
source: "impl-review"
found_during: "itd-130 planning + implementation"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (planning owed: trigger met (abcd update shipped); demote bootstrap.sh to a cold-start trampoline with ownership-checked delegation). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

Demote hooks/bootstrap.sh to a cold-start trampoline once abcd update ships (itd-130 follow-on rung). Trigger: itd-130 shipped. When any abcd binary is resolvable, the hook delegates provisioning to the Go core (abcd update); the raw curl fetch survives only for the no-binary-anywhere cold start — the one state a Go updater cannot serve. Seam hardening carried from itd-130's Decisions: delegate only to a binary passing an ownership check, keep the raw fetch as the fallback when delegation fails, and freeze the release-asset layout as a contract so an old PATH binary provisioning a new plugin root keeps working across cuts. Goal: one canonical downloader instead of three (bootstrap.sh, the README one-liner, the Go verb).