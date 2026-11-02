---
schema_version: 1
id: "iss-2611012124381670"
slug: "abcd-land-record-review-refused-two"
severity: "minor"
category: "ux"
source: "manual-test"
found_during: "recording review pairs with abcd land --record-review, 2026-11-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/landing"
remedy: "Accept a return whose Verdict section names the verdict on a labelled line (Verdict: SHIP) and the head on another, or say in the refusal exactly which line it read and the form it wants: the verdict word alone as the section's first line."
---

abcd land --record-review refused two reviewers' returns whose '### Verdict' section began 'Head: 9d2749a96' and 'Verdict: SHIP': 'its verdict "Head: 9d2749a96" is not one a validator writes; nothing was written'. It accepts only the verdict word alone as the first non-empty line under the heading, which the help text does not say ('each naming the head's first nine characters under one ### Verdict heading'), so a return in an equally clear shape is refused and has to be rewritten by hand.
