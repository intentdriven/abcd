---
schema_version: 1
id: "iss-2609212142568782"
slug: "the-private-banlist-accepts-a-fragment-shorter-than-a-word-and-nothing-warns-that-it-will-match-names"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "abcd lab 3 (lab-260831163412-c3e59af), filed from the capstone handoff on 2026-09-21"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/banlist (add path); the private tier's pattern validation"
remedy: "Waits on the owed ruling (BP, private-name item): in the banlist add path (internal/core/banlist): if warn, print a warning naming the risk when a private pattern has fewer than six literal alphanumerics and no explicit (^|[^[:alnum:]]) boundary; if refuse, refuse such a pattern unless --allow-short is given. Either way the pre-commit refusal names the entry's key and its length class, and a test adds a five-character unbounded pattern and asserts the ruled outcome."
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker: the want is a warning on a short or unbounded private pattern plus an explicit flag to keep it, which leaves open what counts as too short for a regular expression (generated.go's minPhraseAlnum of 3 covers phrases, and the incident was five characters), whether the add warns or refuses, and the new flag on the banlist surface (drain lane drainRest, run A, 2026-09-29)."
---

The private banlist accepts a fragment shorter than a word and nothing warns that it will match names. A five-character fragment in the operator-tier private list matched a cited author's first name, so the guard refused a design branch's merge on one machine until the pattern was refined by hand; the store took the fragment without comment. The pattern itself is private and stays out of the record. Wanted: banlist add warns on a pattern below a declared length or without a word boundary, names the risk (it will match inside ordinary words and names), and takes an explicit flag to keep it; the guard's refusal on a private-tier hit names the pattern's length class so the operator knows where to look.

## Remedy grounds (2026-09-29)

- Why: the owed question is warn or refuse; both are written and neither is picked. The threshold of six sits just above the five-character incident, and the boundary is spelled as a bracket class because the hook matches with grep -iE, where the Perl escape \b is not supported (internal/core/ahoy/defaults/pre-commit).
- Rejected: reusing generated.go's minPhraseAlnum of 3, which the incident's five-character fragment already cleared.
