---
id: dcn-2607140000000016
slug: the-swe-bench-figures-are-now-verified
date: 2026-07-14
---
2026-07-14 (research, CORRECTION — supersedes the two entries above): the SWE-bench
figures are now VERIFIED against OpenAI's primary posts, and one claim is WITHDRAWN.
The "35.5% of audited tasks demanded implementation details never stated" figure DOES
NOT EXIST in the primary — it came from secondary reporting. OpenAI's real taxonomy
for SWE-bench Pro (% of full dataset, agent-flagged / human-flagged, read per-bar from
the chart's aria-labels): overly strict tests 14.4/17.8; low-coverage tests 4.1/9.4;
misleading prompt 6.3/7.5; miscellaneous 1.9/1.2; UNDERSPECIFIED PROMPT 0.6/0.8.
Underspecification is the SMALLEST category (~1%); "overly strict tests" is the largest
by ~20x. CONSEQUENCE: the pitch "abcd's specs-before-code prevents the underspecification
that killed these benchmarks" is aimed at a 1% defect and is DEAD. The surviving claim is
stronger and is what the work now defends: SWE-bench's oracle (a merged PR's tests) is
authored independently of, and after, the task statement, so it can demand what the spec
never said — whereas in abcd THE ACCEPTANCE CRITERIA ARE THE ORACLE, so the oracle cannot
exceed the spec. abcd therefore cannot OVER-reject; it can only UNDER-check (the
low-coverage failure, 4-9%), which compounds with judges over-accepting AI-written code
by up to 1.91x. Claim to defend: "a spec-derived oracle cannot over-reject; it can only
under-check — and under-checking is measurable." Verified figures: SWE-bench Verified
(2026-02-23) 27.6% subset audited, >=59.4% flawed tests, 138 problems x >=6 engineers;
SWE-bench Pro (2026-07-08, retracted) 731-task split, 200 (27.4%) agent-flagged vs 249
(34.1%) human-flagged — the two audits disagree by ~7 points, i.e. automated task-quality
auditing UNDER-DETECTS relative to humans.
