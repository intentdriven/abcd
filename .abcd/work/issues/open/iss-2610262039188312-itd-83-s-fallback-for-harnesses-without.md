---
schema_version: 1
id: "iss-2610262039188312"
slug: "itd-83-s-fallback-for-harnesses-without"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 851 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-83-review-bar-fires-itself.md:66"
remedy: "none (filed automatically)"
related_intents: [itd-83]
finding_ref: cfk-f0301b8fdaac515c
---

itd-83's fallback for harnesses without hooks is a pointer in AGENTS.md's marked section, but the configuration page says a default install writes no marker block

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-83-review-bar-fires-itself.md:66` — "a one-line pointer in the marked conventions section"
- End B: `.abcd/development/brief/05-internals/03-configuration.md:66` — ""skip" is the default, so a default install names abcd"

itd-83 promises the review bar is on once abcd is installed. For a harness that does not run the hook, the fallback is a one-line pointer in the marked conventions section of `AGENTS.md`. The configuration page says `docs.target` defaults to `skip`, so a default install names abcd in no conventions file. On a default install, the fallback has no section to live in.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 851.
