---
schema_version: 1
id: "iss-2610021502084298"
slug: "no-check-meets-the-retired-role-word-in"
severity: "minor"
category: "future-work-seed"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: ruling R5"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/docs-lint.json"
remedy: "Waits on a ruling by the product thinker (R5 did not ask for it): add a record-time check that applies roles/retired-role-word to the Press Release section of every shipped and planned intent, either as a section-scoped root in docs-lint (a whole-file root would also refuse intent bodies, where the word stands in grilled history) or as a site pre-check over the press releases the hero selection can reach, and reword the thirteen press releases still carrying it to the role each sentence means in the same change so the check lands green. Grounds: the site chapter's single-source rule, under which the page is fixed at its source, and itd-2609212137129937's claim that every place abcd writes names which of the two people it means; a press release that passes the check yet fails abcd lint site would show the check wrong."
---

No check meets the retired role word in an intent's press release at the record: the docs-lint rule roles/retired-role-word roots at the docs, the command pages and the rules, so a shipped or planned intent's press release carrying it is refused only when the site composer renders it as the landing page's hero quote, which is how itd-60's first MET audit broke the site-render gate (iss-2610021446271464). After the R5 reword of itd-60, thirteen shipped press releases still carry the word (itd-65, itd-66, itd-67, itd-69, itd-73, itd-74, itd-93, itd-102, itd-111, itd-114, itd-125, itd-2609111003026787, itd-2609212137129937), so the next MET audit ingested for any of them refuses the render again on a branch that only recorded an audit; the product thinker's ruling R5 (2026-10-02) accepted that a future slip breaks the build and did not ask for this check. Related to iss-2610020731591808, which names the brief, the principles and the personas registry as swept but unguarded.
