---
schema_version: 1
id: "iss-2610262036076395"
slug: "the-token-cost-draft-adds-per-session"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 625 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-2609292107351737-every-development-session-records-what.md:21"
remedy: "none (filed automatically)"
related_intents: [itd-2609292107351737]
finding_ref: cfk-9b0b7a135dcf3abc
---

The token-cost draft adds per-session token accounting at capture; 11-history already reports per-session token usage in reconstruct telemetry, with its own counting rule.

A scope leakage (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-2609292107351737-every-development-session-records-what.md:21` — "keeps the per-message token counts and the model id the transcript itself carries"
- End B: `.abcd/development/brief/04-surfaces/11-history.md:168` — "The telemetry file carries the span, turn counts, token usage, a per-tool"

itd-2609292107351737 keeps the per-message token counts and the model id the transcript carries at capture and sums them per session. 11-history's reconstruct already writes a telemetry file carrying the span, turn counts, token usage, per-tool call counts and models per session and per agent, counting tokens once per response id because a per-line sum inflates by 2.44x to 5.28x. Two surfaces claim per-session token cost, and the draft names neither the existing telemetry nor its counting rule, so the two counts can diverge.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 625.
