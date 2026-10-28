---
schema_version: 1
id: "iss-2610262040528498"
slug: "itd-142-writes-interview-grounds-to-a"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 961 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-142-the-brief-creation-interview-abcd.md:36"
remedy: "none (filed automatically)"
related_intents: [itd-142]
finding_ref: cfk-876f6df1c1971089
---

itd-142 writes interview grounds to a bare 'ledger'; the ledger glossary says every ledger but the issue ledger is always named in full.

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-142-the-brief-creation-interview-abcd.md:36` — "receives only confirmed content, a ledger records the grounds"
- End B: `.abcd/development/brief/glossary/core/ledger.md:4` — "Four further ledgers share the word and are always named in full."

itd-142 says the interview's confirmed products go to the brief and 'a ledger' records the grounds, without naming which ledger. The ledger entry keeps bare 'ledger' for the issue ledger and says the other ledgers are always named in full. This one is none of them.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 961.
