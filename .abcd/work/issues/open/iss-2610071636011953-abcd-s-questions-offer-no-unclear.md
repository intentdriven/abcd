---
schema_version: 1
id: "iss-2610071636011953"
slug: "abcd-s-questions-offer-no-unclear"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "product-thinker interviews in an autonomous drain session, 2026-10-07"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/question/asking.go"
remedy: "Give every abcd question a fixed 'Unclear, explain more' option whose answer means: re-ask the same decision with more explanation (a story, one defined term, the consequence of each answer), never record it as a decision; the asking rule and the question check (internal/core/question) require it as an option; because the host's question tool takes at most four options, the check counts it inside the four, so a question offers at most two answers beside 'Decide later' and 'Unclear, explain more', and a decision with three defensible answers is split into two questions."
---

abcd's questions offer no 'Unclear, explain more' answer, though the product thinker needs one often: in one session on 2026-10-07 they typed it by hand into the host's free-text 'Other' field three times ('I don't understand', 'unclear, explain', 'unclear, explain'), and once answered 'what does that have to do with abcd?'. Each time the question was re-asked with more context. The asking rules list exactly the defensible answers plus 'Decide later' or 'None of these', so a person who cannot answer for lack of explanation has only free text to say so, and nothing records how often a question failed to explain itself. Screenshot of the third case: a 'Product Q1' setup question with a typed '4. unclear, explain' entry.
