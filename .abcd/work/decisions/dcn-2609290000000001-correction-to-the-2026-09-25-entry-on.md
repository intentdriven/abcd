---
id: dcn-2609290000000001
slug: correction-to-the-2026-09-25-entry-on
date: 2026-09-29
---
2026-09-29 — Correction to the 2026-09-25 entry on the pre-commit guard's sources refresh (itd-76, iss-2609250834251447): that entry says the scaffolded copy's opt-in shape "answers none of the three questions the ruling holds", and half of that is not so. The template `abcd ahoy` scaffolds (`internal/core/ahoy/defaults/pre-commit`) takes a PROVISIONAL stance on two of the three questions: default or opt-in — opt-in, since nothing runs until the clone sets `abcd.sourcesBinary`; fail open or closed — open, since an unusable opt-in (not an absolute path to an executable regular file) prints one line and the commit proceeds. Only the first question, how a scaffolded hook finds abcd, stays unanswered in the template's own terms, which say it "deliberately does not take" that decision. Both provisional answers stand until the product thinker rules on iss-2609250834251447, and the ruling may replace either. The earlier entry stands as written, since the ledger is append-only (review2-sources finding 5; recorded by lane drainDrift2 of autonomous run A, iss-2609252055533837).
