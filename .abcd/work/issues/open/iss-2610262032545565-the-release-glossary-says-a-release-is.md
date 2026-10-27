---
schema_version: 1
id: "iss-2610262032545565"
slug: "the-release-glossary-says-a-release-is"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 332 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/04-launch.md:10"
remedy: "none (filed automatically)"
finding_ref: cfk-d79dbae3a1563dd8
---

the release glossary says a release is published via launch ship; launch says the verb publishes nothing

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/04-launch.md:10` — "It publishes nothing: the shipped verb writes a dated changelog heading and the release page and stops"
- End B: `.abcd/development/brief/glossary/distribution/release.md:35` — "and for the act of publishing one via `launch ship`"

distribution/release tells the reader to use 'release' for the act of publishing one via launch ship. 04-launch says launch publishes nothing: the shipped verb writes a dated changelog heading and the release page and stops, and CI and a human take it from there. The glossary attributes publication to a verb whose chapter says it never publishes.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 332.
