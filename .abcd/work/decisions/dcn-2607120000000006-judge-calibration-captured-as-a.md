---
id: dcn-2607120000000006
slug: judge-calibration-captured-as-a
date: 2026-07-12
---
2026-07-12 — Judge calibration captured as a DISCIPLINE (itd-81), not a standalone
intent: verdict-rendering agents are plumbing (no user moment), and itd-5 is the
precedent for a cross-cutting rule over agent prompts. Core rule: no judge ships
unmeasured — a labelled corpus with known-good cases ≥40%, scored on true-negative
rate as a first-class metric alongside recall, with a declared TNR floor gating the
prompt lock. Evidence: LLM code judges systematically over-flag and ~1/3 of their
errors are hallucinated code (2603.00539); judges over-rate LLM-written and
under-rate human-written code (2507.16587); ground truth is manufactured by
injecting defects (CriticGPT 2407.00215). CORRECTS itd-5: its pre-flight tiebreak
("passes goldens AND >10% shorter") selects for the brevity bias that ACE
(2510.04618) identifies as destroying instruction quality — struck; the gate is the
corpus score. CONSTRAINS itd-64: reviewer verdicts are not ground truth (the
reviewer is the instrument under measurement) and its tuning loop must stay
human-gated — unattended proxy-optimisation reward-hacks at 73.8% (OpenReview
ikrQWGgxYg). Rejected: judge panels/juries (nine judges → 2.18 effective votes,
correlated errors, no better than the single best judge — 2605.29800); 1-5 severity
scores (middle-drift, position bias); reasoning inside a JSON schema (2408.02442).
