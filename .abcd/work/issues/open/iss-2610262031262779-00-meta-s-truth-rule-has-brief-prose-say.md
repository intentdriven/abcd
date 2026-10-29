---
schema_version: 1
id: "iss-2610262031262779"
slug: "00-meta-s-truth-rule-has-brief-prose-say"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 129 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/00-meta.md:11"
remedy: "Narrow the sentence at brief/README.md:13 to say per-intent and per-spec delivery progress lives in the roadmap dashboard, while brief prose says only whether a described mechanism is shipped or staged, as 00-meta's truth rule requires."
finding_ref: cfk-713dc8a409d0cb26
---

00-meta's truth rule has brief prose say what is shipped and what is staged; the brief README says delivery state lives in the roadmap dashboard and never in brief prose.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/00-meta.md:11` — "variance a reader must resolve is shipped vs staged, and a staged claim says so"
- End B: `.abcd/development/brief/README.md:13` — "lives in the roadmap dashboard, never in brief prose"

00-meta says a section may describe machinery that is shipped or machinery that is staged, and that the only 'variance a reader must resolve is shipped vs staged, and a staged claim says so' explicitly. The brief README says current delivery state 'lives in the roadmap dashboard, never in brief prose'. Shipped against staged is delivery state, so one document requires the brief to carry it and the other forbids it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 129.
