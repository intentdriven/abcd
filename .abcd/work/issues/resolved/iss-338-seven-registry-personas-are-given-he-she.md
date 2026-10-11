---
schema_version: 1
id: "iss-338"
slug: "seven-registry-personas-are-given-he-she"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "bughunt-round-2"
found_at: ".abcd/development/intents"
resolution: "Duplicate of resolved iss-321, which already rewrote all ten cited persona sites (itd-92/97/98/108/126/128/100) to they/them per itd-79; a repo-wide gendered-pronoun sweep over .abcd/ finds zero persona violations remaining. Recorded not-fixed in round 2 before round 1's late-landing iss-321 fix reconciled onto main."
impact: internal
---

Seven registry personas are given he/she/his/her pronouns across intents/drafts (itd-92,97,98,108,126,128) and intents/shipped (itd-100), violating the they/them persona discipline (itd-79); review-enforced convention with no mechanical gate, concentrated in one 2026-08 authoring batch — a cosmetics-batch cleanup

## Evidence

- Registry personas given gendered pronouns: `intents/drafts/itd-92:20`, `itd-97:32`, `itd-98:28`, `itd-108:48`, `itd-126:17,29,31`, `itd-128:23`; `intents/shipped/itd-100:17,25`.
- Rule: `itd-79-persona-registry.md:18`, `personas.json:4`, `intents/README.md:241`, DECISIONS.md -- every persona is they/them.

## Refuter verdict -- CONFIRMED, severity NITPICK (recorded, not fixed)

All ten referents are registry personas (real cited authors correctly excluded); none is a first-person self-description; the narrative attribution clauses sit outside the quotes. No drafts/ exemption exists; itd-100 is shipped/. But comparable persona findings (iss-300, iss-49) are filed nitpick, and no document states the rule wrongly -- only ~7 of ~88 persona-quote files deviate, concentrated in one 2026-08 authoring batch. A natural addition to the iss-49 cosmetics batch; recording is the outcome.
