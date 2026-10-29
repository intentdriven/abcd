---
schema_version: 1
id: "iss-2610262041299335"
slug: "itd-180-gives-the-disposition-record"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1003 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-180-a-cold-reading-s-findings-land-as.md:57"
remedy: "Ruled 2026-10-27, the recommended answer accepted (dcn-2610271112330536): Does the disposition record have four states (accepted, rejected, declined, held) or five? Answers: (a) four, and itd-180's count is a miscount to correct; (b) five, and the fifth is to be named and built. Recommended (a): the code, glossary and spec all ship four and no fifth has been named since 2026-09."
related_intents: [itd-180]
finding_ref: cfk-854f3a6967b8f988
---

itd-180 gives the disposition record five states; the glossary defines a disposition as one of four states

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-180-a-cold-reading-s-findings-land-as.md:57` — "five states, availability varying by position"
- End B: `.abcd/development/brief/glossary/README.md:270` — "in one of four states: accepted, rejected, declined or held."

itd-180 says the disposition record has five states whose availability varies by position, yet it enumerates only accepted, rejected, declined and held. The glossary's term index defines a disposition as one of four states: accepted, rejected, declined or held. The count of states the disposition schema admits cannot be both five and four, and a validator built from the intent's count would look for a state that does not exist.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1003.
