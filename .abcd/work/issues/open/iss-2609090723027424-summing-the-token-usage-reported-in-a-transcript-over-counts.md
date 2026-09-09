---
schema_version: 1
id: "iss-2609090723027424"
slug: "summing-the-token-usage-reported-in-a-transcript-over-counts"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "sub-agent transcript capture conceptual review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/specs/open"
---

Summing the token usage reported in a transcript over-counts it, because the harness repeats one response's usage on every content-block line it writes. A single assistant response is written as several lines, each carrying the same message id and the same usage object, so a naive sum multiplies that response's cost by its block count. Measured on one sub-agent transcript here the naive total was 26.7 million against 15.0 million de-duplicated by message id, a factor of 1.77; measured across ten transcripts the same comparison gave 132,336 against 25,046, a factor of 5.28. The factor varies with how many blocks a response happens to have, so it cannot be corrected after the fact by a constant. Any telemetry derived from these transcripts must state the rule that one usage counts once per message id, and must be tested against a fixture containing a multi-block response with a known total. Without that rule a telemetry file is not merely imprecise, it is confidently wrong by a factor that changes between sessions, which is worse than absent for the comparison across runs that such a file exists to support.
