---
schema_version: 1
id: "iss-2610262037186224"
slug: "itd-107-keeps-forge-calls-arming"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 713 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-107-autonomous-routines-assemble-from-one.md:145"
remedy: "none (filed automatically)"
related_intents: [itd-107]
finding_ref: cfk-74f8ccee9ac5dcf6
---

itd-107 keeps forge calls (arming auto-merge, re-reading created items) out of the binary so it needs no forge command-line tool; the build loop's binary already makes those calls through gh.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-107-autonomous-routines-assemble-from-one.md:145` — "so the binary acquires no platform CLI dependency"
- End B: `.abcd/development/brief/04-surfaces/34-build.md:768` — "It opens the pull request through the forge client the repository already uses (`gh`)"

itd-107 justifies leaving forge calls to the host by saying the binary then acquires no platform CLI dependency. In 34-build's landing, the binary opens the pull request through gh, asks gh pr view whether it merges, arms auto-merge and re-reads the body the forge holds. So the binary already depends on gh, and the draft's reason does not hold.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 713.
