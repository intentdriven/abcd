---
schema_version: 1
id: "iss-2610262035329500"
slug: "the-lint-chapter-says-the-outbound-gate"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 579 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/16-lint.md:91"
remedy: "none (filed automatically)"
finding_ref: cfk-8167fb4d64f8a002
---

The lint chapter says the outbound gate is the fourth surface for its class and the only one that judges text before it is public; the configuration chapter has the shell guard refusing a posting gh pr/issue command over the same shapes.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/16-lint.md:91` — "This is the fourth, and it is the only one that judges text BEFORE it is"
- End B: `.abcd/development/brief/05-internals/03-configuration.md:89` — "the shell guard refuses a posting gh pr/issue command whose title,"

16-lint counts four surfaces that judge the outbound policy's class. It says `abcd lint outbound` is the only one that judges text before it is public. 03-configuration's `outbound.forge` key has the shell guard refuse a posting `gh pr`/`gh issue` command whose title, body or body file carries either shape. That is a further surface, and it judges text before it is posted. So the count of four and the claim 'the only one' cannot both stand.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 579.
