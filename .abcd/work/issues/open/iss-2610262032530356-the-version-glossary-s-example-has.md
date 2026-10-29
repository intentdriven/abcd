---
schema_version: 1
id: "iss-2610262032530356"
slug: "the-version-glossary-s-example-has"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 331 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/04-launch.md:180"
remedy: "Reword the example at version.md line 47 so launch ship derives the version and writes the dated changelog heading and release page, and the release workflow, not the verb, tags v0.2.0."
finding_ref: cfk-c80a09389acb5a87
---

the version glossary's example has launch ship tag the repo; launch says the verb neither commits, tags nor publishes

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/04-launch.md:180` — "verb neither commits, tags, nor publishes"
- End B: `.abcd/development/brief/glossary/distribution/version.md:47` — "`launch ship` stamps the release artifact `0.2.0` (strict SemVer, no leading `v`) and tags the repo `v0.2.0`."

distribution/version's example says launch ship stamps the release artifact and tags the repo v0.2.0. 04-launch says the verb neither commits, tags nor publishes: it writes the dated changelog heading and the release page, and every step past them, the tag included, is performed by a human and by CI. The two disagree on whether launch ship creates the tag.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 331.
