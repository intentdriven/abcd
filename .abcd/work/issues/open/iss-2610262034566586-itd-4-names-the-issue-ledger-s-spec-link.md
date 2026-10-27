---
schema_version: 1
id: "iss-2610262034566586"
slug: "itd-4-names-the-issue-ledger-s-spec-link"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 534 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-4-issue-capture.md:42"
remedy: "none (filed automatically)"
related_intents: [itd-4]
finding_ref: cfk-444d146472d1a82e
---

itd-4 names the issue ledger's spec-link field related_epics; the capture surface page's schema names it related_specs.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-4-issue-capture.md:42` — "`related_epics` (list of `spc-N`)"
- End B: `.abcd/development/brief/04-surfaces/06-capture.md:417` — "related_specs: [spc-N, ...]"

itd-4 lists the issue frontmatter field holding spec links as `related_epics`. 06-capture.md gives the schema that the writer and the committed-ledger gate share, and there the field is `related_specs`. Two names for one field: a record written to itd-4's schema carries a key the shipped schema does not define.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 534.
