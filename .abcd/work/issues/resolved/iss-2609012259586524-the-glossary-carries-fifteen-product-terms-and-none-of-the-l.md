---
schema_version: 1
id: "iss-2609012259586524"
slug: "the-glossary-carries-fifteen-product-terms-and-none-of-the-l"
severity: "major"
category: "documentation"
source: "agent-finding"
found_during: "iteration-2 conformance audit against the design framework v4 and the readings companion v4, 2026-09-01"
origin: researcher-authored
production_mode: dictated-and-formatted
found_at: ".abcd/development/brief/glossary"
resolution: "The ledger bounded context lands in this change with nine terms as they presently stand, under adr-55's admissibility rule."
impact: additive
---

The glossary carries fifteen product terms and none of the ledger's own vocabulary, so the widening reading reads against terms that do not name its object

The readings companion (v4, section 5.6) states the widening reading's bound in
advance: "The construal is one or two sentences and the glossary three to six
terms." The glossary is one of the four surfaces the widening reading is given
(brief current text with the construal, the glossary, the disciplines, the
specs) and the companion's section 5.4 requires every returned configuration to
name "the element of the stated construal, vocabulary or scope under which the
configuration falls."

The committed glossary under `.abcd/development/brief/glossary/` holds fifteen
terms in three groups: core (lifeboat, voyage, oracle, disembark, transport,
phase, intent, spec, brief, persona), distribution (end-user, version, release)
and interview (session, embark). None of them is a term of the ledger the
construal describes. Cold, warm, construal, reading, position, regime,
disposition, admission, surprise, lapse and the read block are all used
throughout the record and the four definitions, and none is committed as
vocabulary.

Two consequences follow. The widening reading is asked to name the vocabulary
element that admits each configuration, and the vocabulary it is handed does
not contain the words the construal is written in. And adr-55 places the
discipline of the glossary on whoever writes it, with a breach being a lapse-log
entry: the glossary is the one framing surface a reading sees, so its terms are
the construal's terms or they are noise.

The remedy is a record act rather than code: commit the ledger's vocabulary as
glossary terms under adr-55's admissibility rule (the term as it presently
stands, never the dispute that settled it), and state the bound the companion
asks for. Whether fifteen product terms should also travel to the widening
reading, against a bound of three to six, is a scope question the presets can
answer once the terms exist.

## Grounds

- pursued: we expect committed ledger vocabulary to let a widening reading name the vocabulary element that admits each configuration, which the companion requires of every returned item; a widening run whose items cannot cite a committed term would show the terms were the wrong ones
