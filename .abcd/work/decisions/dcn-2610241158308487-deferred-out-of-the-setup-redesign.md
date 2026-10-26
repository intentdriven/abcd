---
id: dcn-2610241158308487
slug: deferred-out-of-the-setup-redesign
date: 2026-10-24
---
2026-10-24 — Deferred out of the setup redesign: iss-2610241157493381 (the session-end retry test fails under machine load because its retry budget is wall-clock). It lies outside the redesign's footprint (internal/surface/cli/hook_subagent.go, history capture), fails on an unchanged branch, and is recorded so it is fixed on its own rather than stepped over; the redesign's checks report it as a known failure, not a pass.
