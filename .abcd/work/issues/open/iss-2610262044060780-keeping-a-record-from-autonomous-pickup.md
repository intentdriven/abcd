---
schema_version: 1
id: "iss-2610262044060780"
slug: "keeping-a-record-from-autonomous-pickup"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1183 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-57-manual-hold-sentinel.md:17"
remedy: "none (filed automatically)"
related_intents: [itd-57, itd-2609201916151817]
finding_ref: cfk-a39a4061facc3c8d
---

Keeping a record from autonomous pickup is a sentinel-spec dependency in itd-57 and a held state the build loop checks in itd-2609201916151817

A scope leakage (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-57-manual-hold-sentinel.md:17` — "a permanent sentinel spec, scaffolded at configure time, that any future spec can depend on to stay invisible to autonomous pickup"
- End B: `.abcd/development/intents/planned/itd-2609201916151817-one-verb-takes-a-single-intent-from.md:43` — "The held state (`iss-2609200830076665`) is a prerequisite this intent depends on"

itd-57 proposes holding work back from autonomous pickup with a dependency on a permanent sentinel spec, which the selector already skips. itd-2609201916151817 makes 'not held' a readiness check and depends on a separate held state (iss-2609200830076665). The same ground, a person keeping an item out of an unattended run, gets two incompatible mechanisms, and neither record mentions the other.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1183.
