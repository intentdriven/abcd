---
schema_version: 1
id: "iss-2610262038399413"
slug: "itd-1-says-the-ship-needs-work-major"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 809 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-1-acceptance-gates.md:47"
remedy: "none (filed automatically)"
related_intents: [itd-1]
finding_ref: cfk-f72f62ab428645ba
---

itd-1 says the SHIP/NEEDS_WORK/MAJOR_RETHINK review verdicts score plan and implementation reviews; 01-agents limits them to a packed lifeboat

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-1-acceptance-gates.md:47` — "which score *changes/runs* (oracle reviews of plans, implementations, completions; consumed by the native receipt schema validator)"
- End B: `.abcd/development/brief/05-internals/01-agents.md:222` — "`{SHIP, NEEDS_WORK, MAJOR_RETHINK}` — assess a packed *lifeboat*."

itd-1 says the review-verdict family scores changes and runs: oracle reviews of plans, implementations and completions. 01-agents, which itd-1 cross-references, says that family assesses a packed lifeboat only, and that the implement loop's reviewers use words of their own. So 'review verdict' covers two different scopes, and a builder cannot tell which verdicts a plan or implementation review returns.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 809.
