---
schema_version: 1
id: "iss-2610262032307333"
slug: "launch-s-citation-baseline-row-refuses"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 277 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/04-launch.md:338"
remedy: "none (filed automatically)"
finding_ref: cfk-43836ca0dc59ebb6
---

launch's citation-baseline row refuses on an overdue citation; docs says the launch preflight only names what would stop a release, while the release still cuts

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/04-launch.md:338` — "against their receipts and refuses on a broken, unreceipted or overdue one."
- End B: `.abcd/development/brief/04-surfaces/10-docs.md:228` — "which names what the citation baseline would stop a release on while a release still cuts."

04-launch.md says the citation-baseline row of the pre-flight suite refuses on a broken, unreceipted or overdue citation. 10-docs.md says launch computes its own citation preflight and that this preflight names what the citation baseline would stop a release on, while a release still cuts. In docs, only the lint's release-gate flag, passed by the release workflow, makes an overdue citation a blocker, and no committed config can arm it. Either the cut refuses on an overdue citation without that flag, or it still cuts.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 277.
