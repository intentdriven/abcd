---
schema_version: 1
id: "iss-2610262036597501"
slug: "the-user-scope-inventory-has-no"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 689 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md:621"
remedy: "none (filed automatically)"
finding_ref: cfk-35d393e3f9db697e
---

The user-scope inventory has no dashboard store, while the dashboard chapter keeps its run file and devices list in `~/.abcd.noindex/dashboard/`

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/03-configuration.md:621` — "one per machine, machine-local shared state only: the history registry, the transcript corpus, the voyage operations namespace, the lab store"
- End B: `.abcd/development/brief/04-surfaces/36-dashboard.md:127` — "The run file and the devices list live in `~/.abcd.noindex/dashboard/` and name no project content."

03-configuration lists, store by store, what the user scope holds and says the ahoy tree must agree with that list. No `dashboard/` store is on it. 36-dashboard writes its run file and the list of devices that opened it under `~/.abcd.noindex/dashboard/`. Either the inventory is incomplete or the dashboard keeps a store the inventory does not admit.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 689.
