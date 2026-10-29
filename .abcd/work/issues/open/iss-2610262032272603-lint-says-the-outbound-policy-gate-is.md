---
schema_version: 1
id: "iss-2610262032272603"
slug: "lint-says-the-outbound-policy-gate-is"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 271 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/01-ahoy.md:324"
remedy: "In 16-lint.md, outbound-policy section: replace 'the fourth, and the only one that judges text BEFORE it is public' with five surfaces, saying the shell guard's forge-text-outbound entry also runs scanner.CheckOutbound on a gh pr or gh issue command before it posts."
finding_ref: cfk-da141d4467e0243e
---

lint says the outbound-policy gate is the only surface that judges text before it is public; ahoy says the shell guard also refuses forge text carrying the same two shapes before it is posted

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/01-ahoy.md:324` — "so the shell guard also refuses a posting `gh pr` or `gh issue` command whose title, body or body file carries either shape"
- End B: `.abcd/development/brief/04-surfaces/16-lint.md:91` — "This is the fourth, and it is the only one that judges text BEFORE it is public."

16-lint.md counts four surfaces that read the harness-leak definition and says the outbound gate is the only one that judges text before it is public. 01-ahoy.md says the outbound opt-in also makes the shell guard refuse a posting gh pr or gh issue command whose title, body or body file carries either shape (forge-text-outbound). That is a judgement of text before it is public. Both cannot hold: either the guard's forge-text check is a fifth pre-public surface, or lint's count and its 'only one' claim are wrong.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 271.
