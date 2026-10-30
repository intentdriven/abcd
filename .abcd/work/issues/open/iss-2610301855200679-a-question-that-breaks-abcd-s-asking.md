---
schema_version: 1
id: "iss-2610301855200679"
slug: "a-question-that-breaks-abcd-s-asking"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "planning interview of the spot-check draft"
origin: researcher-authored
production_mode: hand-written
found_at: "agents/question-drafter.md"
remedy: "Turn the question-drafter into a question-writing agent that keeps working until its draft passes abcd's own question check (the read-only check itd-2610230756259215 proposes) and only then returns it: draft, check, fix every fault, repeat; never a refusal for length, only for a missing part of the brief. Its layout rules are configurable per repository, with abcd's rules as the default; for example, when the material or the options are too long for one screen, it splits the question into numbered parts (Q1a, Q1b), each a checked question. The calling agent puts to the person only a draft the check passed and never writes or edits a question itself; the hook's refusal stays the last resort, not the first check."
refines: [itd-2610230756259215, iss-2610230745597968]
related_intents: [itd-2610230756259215]
---

A question that breaks abcd's asking rules still reaches the person as an error, after the product thinker asked several times for it to stop (2026-10-30, and before in iss-2610230745597968). In one planning interview the abcd:question-drafter returned a refusal for 3 of 10 drafts because the quoted material and its options did not fit the 24-row limit for one decision; the calling agent then put over-limit questions to the person itself, and once skipped the drafter altogether, so the question check blocked the question in front of the person (no Decide later option, 28 rows). Three gaps let this through: the drafter may give up and hand back a refusal instead of working until its draft passes; it counts rows by its own reckoning rather than through abcd's check; and nothing makes the calling agent ask only a question that passed, so the hook's refusal is the first check the question meets, and the person sees it. The product thinker's proposal: a sub-agent keeps working on the question until it conforms, and only then is it put to the person.

## Refinement 2026-10-30

The product thinker refined the proposal the same day: this is a question-writing agent whose rules can be configured where a repository needs it, and a question too long for one screen is split into numbered parts, such as Q1a and Q1b, rather than shortened or sent over the limit.
