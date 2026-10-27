---
schema_version: 1
id: "iss-2610262045017395"
slug: "itd-102-records-a-canonical-project"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1246 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-102-your-repo-says-the-same-thing-about.md:17"
remedy: "none (filed automatically)"
related_intents: [itd-102, itd-2610031348087517]
finding_ref: cfk-066f7dafa55c631e
---

itd-102 records a canonical project title that surfaces derive from; itd-2610031348087517 says nothing in a project derives a title reliably

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-102-your-repo-says-the-same-thing-about.md:17` — "interviews the maintainer for the project's identity — its title, its tagline, and a short elevator pitch"
- End B: `.abcd/development/intents/shipped/itd-2610031348087517-when-abcd-sets-up-a-project-it-asks-the.md:44` — "abcd never writes a title (nothing in a project derives one reliably)"

itd-102 interviews for the project's title, tagline and pitch, records them in one canonical home in the development record, and derives every rendered surface (README strapline included) from it. itd-2610031348087517 leaves the title out of the README parts abcd keeps, on the ground that nothing in a project derives one reliably. The identity block itd-102 establishes is exactly such a source inside the project. The second record's ground contradicts the first record's premise.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1246.
