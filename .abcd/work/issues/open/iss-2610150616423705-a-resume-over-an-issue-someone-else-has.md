---
schema_version: 1
id: "iss-2610150616423705"
slug: "a-resume-over-an-issue-someone-else-has"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "final fix round of spc-2610031753359849 steps 12, 13 and 15 (feat/reserve-build)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/reservation.go"
remedy: "Release, rather than renew or reserve, a resumed run's reservation whose issue has reached resolved/ or wontfix/."
---

A resume over an issue someone else has since resolved or wontfix'd renews or reserves a reservation on a closed issue, which then holds until it lapses: reserveResumed reads found_at from whichever status folder holds the issue and reserves it again. Harmless, since no second holder can appear, but origin shows a closed issue as reserved.
