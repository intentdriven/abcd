---
schema_version: 1
id: "iss-2611011402496418"
slug: "a-planning-interview-run-in-a-plain"
severity: "major"
category: "bug"
source: "manual-test"
found_during: "first planning interview on the API-key runner, 2026-11-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/interview/written.go"
remedy: "Hand a question, or a done outcome, that abcd's check refuses back to the interviewer with the refusal in its next brief, so it redrafts without the person seeing it; make the rows limit a warning on the terminal route, drawing the question and telling the role to keep the next within it, as the host's question check already does (question.RefusingFinding); bound consecutive redrafts by a repository setting, default 3, adding its key to the interview claim in internal/core/layered/interview.go, which today refuses every key but interview.list; at the bound, show the refused question as it stands, repairing mechanically what the refusing rules allow (the decide-later option, bold and recommendation marks, over-long labels), and for a done outcome still refused at the bound end with a plain message naming what happened and what to run next, never the internal finding; keep every model text shown passing through question.Safe; update the brief's 'the interview stops' line and TestAnAskFailingTheCheckIsAnInvalidFallback and TestAnAskBreakingTheAskingLimitsIsInvalid."
---

A planning interview run in a plain terminal on the API-key runner ended at its first question on 2026-11-01. The interviewer wrote a sound question, but abcd's question check refused it at 39 rows against a 32-row limit, and the refusal was treated as the runner's invalid answer: with no host fallback in a terminal the interview stopped, printing 'runner: claude invalid for planning-interviewer ... the contract's validator refused its answer: the question breaks 1 rule(s): tab 1, whole tab (rows)'. The person was left with an internal message and no next step. The brief tells the role 'abcd refuses a receipt ... whose question breaks a rule below, and the interview stops', and only a refused done outcome is ever handed back for another turn. A host session's own question check shows an over-long question with a warning and has the agent redraft refused ones silently, so the terminal route is stricter than the host route for the one rule a model cannot count exactly. The product thinker ruled the same day that stopping an interview is never acceptable.

2026-11-01: the product thinker ruled the bound and the end state (dcn recorded the same day): redrafts are bounded by a per-repository setting, three by default, and at the bound abcd shows the question as it stands, repaired mechanically where it can be, instead of ending the interview.

2026-11-01, second occurrence: the product thinker retried a planning interview of itd-2610262140595217 at 21:45 on the main key from a plain terminal, and it ended before any question was shown, the check refusing the first question at 39 rows against 32 ("question text 23, options 8"), with the same internal message. The terminal route has now failed both of the product thinker's attempts on the key on this one rule.
