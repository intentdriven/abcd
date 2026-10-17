---
id: dcn-2607110000000001
slug: iss-35-s-brief-surface-cross-check-is
date: 2026-07-11
---
2026-07-11 — iss-35's brief↔surface cross-check is **bidirectional, but only
the structural half is deterministically lintable**: Direction B (every
`commands/`+`skills/` entry has a brief home) is a coverage lint like
`directory_coverage`; Direction A (brief claims match *binary behaviour* —
flags, exit codes, schema fields, counts) is irreducibly semantic and stays an
LLM/agent job (encoding binary facts into the linter just moves the drift).
So "graduate the detector to a record-lint rule" is a *reshaping* (extract the
deterministic half; keep the semantic half as a periodic/agent check), not a
port. The graduation is a design gate held for maintainer sign-off — options in
`.abcd/development/plans/2026-07-11-iss35-record-lint-graduation.md` (recommend
Option A, structural `surface_coverage` rule) — and it is **blocked** until the
docs/history surface-taxonomy adjudication is decided (a coverage rule fires on
the three chapterless shipped verbs the moment it is armed).
