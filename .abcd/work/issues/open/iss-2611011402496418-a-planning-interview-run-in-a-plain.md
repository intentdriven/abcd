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
remedy: "Hand a question, or a done outcome, that abcd's check refuses back to the interviewer with the refusal in its next brief, so it redrafts without the person seeing it; make the rows limit a warning on the terminal route, drawing the question and telling the role to keep the next within it, as the host's question check already does; bound the consecutive redrafts by a repository setting (interview.redraft_limit in .abcd/config.json, default 3); at the bound, show the question as it stands, repairing mechanically what can be (supply the missing decide-later option, strip bold and recommendation marks, trim over-long labels) and, where a refusing finding cannot be repaired, take a free-text answer to the question's text instead of ending; strip terminal control sequences from any model text shown; update the brief's 'the interview stops' line and TestAnAskFailingTheCheckIsAnInvalidFallback and TestAnAskBreakingTheAskingLimitsIsInvalid."
---

A planning interview run in a plain terminal on the API-key runner ended at its first question on 2026-11-01. The interviewer wrote a sound question, but abcd's question check refused it at 39 rows against a 32-row limit, and the refusal was treated as the runner's invalid answer: with no host fallback in a terminal the interview stopped, printing 'runner: claude invalid for planning-interviewer ... the contract's validator refused its answer: the question breaks 1 rule(s): tab 1, whole tab (rows)'. The person was left with an internal message and no next step. The brief tells the role 'abcd refuses a receipt ... whose question breaks a rule below, and the interview stops', and only a refused done outcome is ever handed back for another turn. A host session's own question check shows an over-long question with a warning and has the agent redraft refused ones silently, so the terminal route is stricter than the host route for the one rule a model cannot count exactly. The product thinker ruled the same day that stopping an interview is never acceptable.

2026-11-01: the product thinker ruled the bound and the end state (dcn recorded the same day): redrafts are bounded by a per-repository setting, three by default, and at the bound abcd shows the question as it stands, repaired mechanically where it can be, instead of ending the interview.
