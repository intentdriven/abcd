---
schema_version: 1
id: "iss-2610262038247394"
slug: "the-surfaces-register-keeps-grill-as-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 792 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/README.md:393"
remedy: "none (filed automatically)"
finding_ref: cfk-79d9ffc3e6c4601e
---

The surfaces register keeps grill as a live design target to become an `intent` sub-verb; the build sequence says itd-94's planning interview supersedes it.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/README.md:393` — "`/abcd:grill` was likewise proposed as a skill. Its design target is promotion to"
- End B: `.abcd/development/brief/06-delivery/01-build-sequence.md:101` — "under itd-94, which supersedes it"

04-surfaces/README says `/abcd:grill`'s design target is promotion to a sub-verb of `/abcd:intent`. 01-build-sequence says there is no grill sub-verb because the grilling itd-27 asked for is the planning interview, `abcd intent interview`, under itd-94, which supersedes it. A superseded design is not a pending one.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 792.
