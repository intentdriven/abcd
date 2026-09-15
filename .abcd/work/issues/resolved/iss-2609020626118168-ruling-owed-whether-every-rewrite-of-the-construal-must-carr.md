---
schema_version: 1
id: "iss-2609020626118168"
slug: "ruling-owed-whether-every-rewrite-of-the-construal-must-carr"
severity: "minor"
category: "architectural-insight"
source: "agent-finding"
found_during: "iteration-2 planning, decomposition of the cold-reading intents, 2026-09-02"
origin: researcher-authored
production_mode: dictated-and-formatted
found_at: ".abcd/development/brief/01-product/06-framing.md"
resolution: "Adopted by the maintainer on 2026-09-02 as an ADR in this change, in the form that fingerprints the three frame surfaces adr-55 enumerates, not the single-section reading this record took first; the rule over every frame edit stays unadopted."
impact: internal
---

Ruling owed: whether every rewrite of the construal must carry a reframe record, which refines adr-55 and reaches rewrites no reading occasioned

The record-discipline review of the Iteration 2 intents (2026-09-02) routed this
question out of the reframe-record intent under the decompose-before-filing
rule. That intent records a reframe occasioned by a reading, joined to the
reading item, disposition or surprise that occasioned it, with the construal's
committed hash before and after.

The open question is wider than a reading. adr-55 rules that the construal as it
presently stands is committed record and that its history stays on the local
side; the framing chapter says the section is rewritten by delivery like any
other brief passage. If every rewrite of the construal, whatever occasioned it,
must carry a reframe record, that is a standing rule over the brief and a lint
over its edits, and it refines adr-55 by adding a committed pointer to an event
whose content adr-55 keeps local. If only reading-occasioned rewrites carry one,
the record shows which reframes a reading caused and nothing about the rest.

Either answer is defensible and the choice is the maintainer's: the first makes
reframes countable at the cost of a rule on every construal edit; the second
keeps adr-55 as it stands and leaves the researcher's own reframes unrecorded.
The reframe intent takes the second reading until this is ruled.

A second, narrower question rides with it. adr-55 counts the framing
section's statement, the glossary's committed terms and the committed scope as
the construal as it presently stands; the reframe record keys on the construal
section alone, so a rewrite of the glossary or of committed scope is not a
reframe by its definition. Either the frame is ruled to be the construal
section and adr-55's wording is narrowed to match, or the record hashes all
three surfaces. The intent takes the first reading and flags it.

## Grounds

- pursued: we expect a decision record adopted before the intent that waits on it to keep the trust rule out of the intent's scope list, which the decompose-before-filing rule requires; an intent shipping ahead of its ADR would show it wrong
