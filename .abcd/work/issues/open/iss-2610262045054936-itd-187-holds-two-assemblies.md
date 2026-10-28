---
schema_version: 1
id: "iss-2610262045054936"
slug: "itd-187-holds-two-assemblies"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1251 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-187-amnesia-is-a-repository-property-proven.md:29"
remedy: "none (filed automatically)"
related_intents: [itd-187, itd-2609020625402599]
finding_ref: cfk-2b5ac623e5c58be1
---

itd-187 holds two assemblies byte-identical with only the manifest excluded; itd-2609020625402599 puts a per-run stamp naming the run in every bundle

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-187-amnesia-is-a-repository-property-proven.md:29` — "byte-equality of the assembled input, manifest excluded."
- End B: `.abcd/development/intents/shipped/itd-2609020625402599-the-scribe-s-context-is-assembled-and.md:21` — "Every assembly, reading or scribe, carries a per-run stamp naming its kind and its run"

itd-187 states the amnesia identity relation as byte-equality of the assembled input with the manifest excluded, so one state assembled twice must be byte-identical. itd-2609020625402599 makes every reading assembly carry a per-run stamp naming its kind and its run. Two assemblies are two runs, so their assembled inputs differ in that stamp. The relation itd-187 states fails unless a second exclusion is added, and itd-187 does not name one.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1251.
