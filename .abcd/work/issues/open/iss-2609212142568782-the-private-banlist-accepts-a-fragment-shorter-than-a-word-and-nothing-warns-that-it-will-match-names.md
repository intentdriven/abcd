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
---

The private banlist accepts a fragment shorter than a word and nothing warns that it will match names. A five-character fragment in the operator-tier private list matched a cited author's first name, so the guard refused a design branch's merge on one machine until the pattern was refined by hand; the store took the fragment without comment. The pattern itself is private and stays out of the record. Wanted: banlist add warns on a pattern below a declared length or without a word boundary, names the risk (it will match inside ordinary words and names), and takes an explicit flag to keep it; the guard's refusal on a private-tier hit names the pattern's length class so the operator knows where to look.
