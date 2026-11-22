---
id: dcn-2611211505053314
slug: decided-by-the-run-for-owner-review-on
date: 2026-11-21
---
2026-11-21 — Decided by the run for owner review, on the owner's delegation of 21 Nov 2026 as part of the autonomous-run experiment: the run records each private advisory's triage verdict itself, with its justification, under --by naming the run and the delegation, instead of queueing the verdicts. It fixes every advisory triaged a vulnerability and, once all are fixed, a release is cut. The asymmetric review bar of dcn-2611211501156446 still holds: a closing verdict is recorded only when two independent reviewers agree, and a split is left untriaged for the owner. Closing or publishing an advisory on the forge (embargo close and publish, each with --approve) is not part of the delegation and stays queued for the owner. To reverse: re-triage any verdict (a second triage keeps the first in its history).
