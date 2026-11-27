---
schema_version: 1
id: "iss-2611262200106745"
slug: "abcd-land-record-review-refuses-a-review"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/landing/review.go"
remedy: "Count a line as a second verdict only when the word verdict is followed, on that line, by one of the known verdict words (SHIP, FIX FIRST, BLOCK, APPROVE, NEEDS-INPUT), so a reference such as 'verdict:52' or a sentence about a verdict record is not one; test both a reference bullet (accepted) and a 'Verdict: SHIP' line outside the heading (still refused)."
resolution: "the recorder reads a word followed directly by a line number or the rest of a file name as a file reference, not a verdict, unless a verdict word follows on the same line (internal/core/landing/review.go, fileReference and statesVerdictWord)"
impact: fix
---

abcd land --record-review refuses a review return whose finding bullets lead with the word verdict as a reference into the record under review, not as a verdict: a security return for an ideate verdict record opened two finding bullets with a reference into that record ('verdict:52', 'verdict:143'), and the recorder counted three Verdict sections and refused it ('it has 3 Verdict sections; one states the verdict, so reword the line ...'). The reviewer had to rename every reference to 'record:N' by hand (2026-11-26, ideate-terminal-build-1126). Any review that cites an ideate verdict record, or a file named after verdicts (verdict.md:14, verdict_head_test.go), leads a line this way; a sentence about a verdict was already accepted.
