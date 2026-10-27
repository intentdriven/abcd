---
schema_version: 1
id: "iss-2610262041393479"
slug: "itd-201-says-the-grill-domain-and-the"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1015 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-201-every-question-abcd-s-agents-put-to-a.md:79"
remedy: "none (filed automatically)"
related_intents: [itd-201]
finding_ref: cfk-08766fa88e483833
---

itd-201 says the GRILL domain and the interview page carry the knowledge-floor text; the knowledge-floor glossary page says GRILL only points to it and the page states the floor in full

A scope leakage (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-201-every-question-abcd-s-agents-put-to-a.md:79` — "The GRILL domain and the interview page carry the text."
- End B: `.abcd/development/brief/glossary/interview/knowledge-floor.md:19` — "The asking rules every abcd interview follows (the GRILL rule domain) point here in one line,"

itd-201's decision on the knowledge floor says the GRILL rule domain and the interview page carry the text. The knowledge-floor entry, which names itd-201 as its origin, says the asking rules point to it in one line and that it states the floor in full. The full statement of the floor then has two claimed homes, and an edit to one will not reach the other.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1015.
