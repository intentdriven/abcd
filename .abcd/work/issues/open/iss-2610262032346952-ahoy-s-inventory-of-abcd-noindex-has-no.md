---
schema_version: 1
id: "iss-2610262032346952"
slug: "ahoy-s-inventory-of-abcd-noindex-has-no"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 288 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/01-ahoy.md:420"
remedy: "none (filed automatically)"
finding_ref: cfk-0879a000841522cb
---

Ahoy's inventory of ~/.abcd.noindex has no dashboard/ entry, but the dashboard keeps its run file and device list there

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/01-ahoy.md:420` — "~/.abcd.noindex/ USER SCOPE — one per machine (machine-local state only)"
- End B: `.abcd/development/brief/04-surfaces/36-dashboard.md:127` — "The run file and the devices list live in `~/.abcd.noindex/dashboard/`"

01-ahoy lists the user-scope directory entry by entry as the machine-local state abcd keeps, and says this inventory and the configuration chapter's are one list that must agree. 36-dashboard puts its run file and device list in ~/.abcd.noindex/dashboard/, which the inventory does not include. Either the inventory is incomplete, or the dashboard writes state outside what abcd says it keeps.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 288.
