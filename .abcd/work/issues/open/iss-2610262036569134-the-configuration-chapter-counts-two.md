---
schema_version: 1
id: "iss-2610262036569134"
slug: "the-configuration-chapter-counts-two"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 685 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md:661"
remedy: "none (filed automatically)"
finding_ref: cfk-1df57a93e1507cd9
---

The configuration chapter counts two limits in load-limits, while the implement chapter names three keys in that file

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/03-configuration.md:661` — "the load check's two limits in `load-limits` (read-only and never created, itd-2609231434459890)"
- End B: `.abcd/development/brief/04-surfaces/27-implement.md:253` — "The limits live in the caller's machine tier, `~/.abcd.noindex/load-limits` (`stray-minutes`, default 30; `extreme-load`, default four times the online core count; `wait-load`, default twice it)"

27-implement lists three keys in `~/.abcd.noindex/load-limits`: stray-minutes, extreme-load, and wait-load, the limit the wait gate opens below. The user-scope inventory in 03-configuration describes the same file as holding the load check's two limits. A reader building the file from the configuration chapter would leave out the wait limit the gate depends on.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 685.
