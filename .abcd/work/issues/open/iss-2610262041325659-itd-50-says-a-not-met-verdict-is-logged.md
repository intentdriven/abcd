---
schema_version: 1
id: "iss-2610262041325659"
slug: "itd-50-says-a-not-met-verdict-is-logged"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1007 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-50-loop-toward-acceptance.md:25"
remedy: "none (filed automatically)"
related_intents: [itd-50]
finding_ref: cfk-8911bd5434821fa9
---

itd-50 says a NOT_MET verdict is logged and "the voyage moves on", using voyage for project progress; the glossary reserves voyage for the operations namespace

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-50-loop-toward-acceptance.md:25` — "records a `NOT_MET` verdict and the voyage moves on"
- End B: `.abcd/development/brief/glossary/core/voyage.md:59` — "Do not use it for a "project" (too generic)"

itd-50's press release uses voyage for the project's onward progress after an audit. The glossary defines voyage only as the operator-level record of disembark and embark runs, and says not to use it for a project. adr-35 retired that lifecycle sense.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1007.
