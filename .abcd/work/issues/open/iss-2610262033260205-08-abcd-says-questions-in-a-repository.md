---
schema_version: 1
id: "iss-2610262033260205"
slug: "08-abcd-says-questions-in-a-repository"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 404 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/08-abcd.md:184"
remedy: "none (filed automatically)"
finding_ref: cfk-e2375669e4f7f51b
---

08-abcd says questions in a repository abcd does not manage are not gated, but 17-guard checks abcd's questions against the field limits whether the repository is managed or not.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/08-abcd.md:184` — "questions are not gated."
- End B: `.abcd/development/brief/04-surfaces/17-guard.md:137` — "`question.Default`) wherever the hook runs, managed or not: the setup interview"

08-abcd says a repository abcd does not manage, or one without the local tier, has no badge and its questions are not gated. 17-guard says abcd's question is held to the field limits wherever the hook runs, managed or not, because the setup interview asks its questions before a repository is managed. Under 17-guard a limit finding refuses the question with the host's deny, so only the badge and mode part is skipped in such a repository.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 404.
