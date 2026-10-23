---
id: dcn-2610180720397970
slug: the-claim-s-generation-number-is-its-tip
date: 2026-10-18
---
2026-10-18 — The claim's generation number is its tip commit (facilitator-on-delegation, itd-2610170634394993 planning prep): every act goes through a fast-forward push, which is already a compare-and-swap, and a take-over moves the tip, so no separate counter is added; the forge enforces the fence only on a push, so for queueing and tagging the criterion says abcd refuses the act, and whether one atomic push can carry the work branch and the renewal together is left to probe at planning (grounds: internal/core/claim/claim.go; internal/core/implement/loop/reservation.go; Chubby section 2.4; git-scm.com/docs/git-push --atomic)
