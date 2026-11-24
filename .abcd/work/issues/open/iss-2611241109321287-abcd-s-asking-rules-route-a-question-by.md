---
schema_version: 1
id: "iss-2611241109321287"
slug: "abcd-s-asking-rules-route-a-question-by"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "surge planning interview, 24 November"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/question"
remedy: "Make the question check in abcd guard hook read facilitator.mode: under auto, refuse a Tech-chip question unless it names an act reserved to a person or is marked critical, and say to decide and record it instead; refuse a Product-chip question whose material names technical surfaces (gates, CI, queue, worktrees, advisories, code names); and add a generalist-level summary requirement for Tech questions outside auto."
---

abcd's asking rules route a question by chip (Product or Tech) but never by the repository's facilitator mode, so under facilitator.mode auto a session still puts technical questions to the person, and nothing stops technical detail reaching the product thinker. On 2026-11-24 a dependency-bump question, a worktree-removal question and an advisory-text question all reached the person in a repository set to auto, against dcn-2611241109230165.
