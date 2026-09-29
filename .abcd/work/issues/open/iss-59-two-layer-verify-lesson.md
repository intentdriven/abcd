---
schema_version: 1
id: "iss-59"
slug: "two-layer-verify-lesson"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "autonomous-run"
found_at: ".abcd/development/roadmap/phases/phase-5-run-seam.md"
remedy: "Waits on ruling E: if a principle, add one under `.abcd/development/principles/` stating that a pattern check proves absence, never truth, and a corpus sweep is done only after a judgement review passes, proven by record-lint's principle rules; if a run-seam gate, carry it into itd-2609201916151817 (which superseded itd-29) as a criterion that `implement check land` refuses a lane declared a corpus sweep with no logged review event, proven by a test that such a land is refused."
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25; rulings-owed E): Does grep-verify-is-not-truth become a principle, or a run-seam gate?"
---

Two-layer verification lesson from the iss-36 sweep: 73 mechanical fixers + per-file grep-verify reported 100% clean, yet a 4-segment judgement review found ~30 real truth defects (invented homes for unmigrated material, false present-tense capability claims). For the run seam (itd-29/Phase 5) and any future corpus sweep: grep-verify proves pattern absence, never truth; a sweep is done only after a judgement review layer passes. Also: fix agents given precise per-finding instructions applied 100% cleanly — precision of the fix list, not agent capability, was the quality lever.

## Remedy grounds (2026-09-29)

- itd-29, the run-seam home the record names, is superseded by the planned itd-2609201916151817, so the gate option lands there; the implement log already records a review event a land check can read.
- No outside-practice check: the lesson is this repository's own measured sweep (73 fixers, about 30 truth defects).
- Rejected: a stricter grep, which proves absence more thoroughly and truth no better.
