---
schema_version: 1
id: "iss-2610262033277086"
slug: "the-build-s-landing-runs-the-spec-close"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 407 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md:518"
remedy: "none (filed automatically)"
finding_ref: cfk-cc6323a3060cef58
---

The build's landing runs the spec close as an automatic stage, but the close refuses without an impact judgement that no build check requires and no stage supplies.

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/05-intent.md:518` — "the impact is demanded at the close that ships and refused at any earlier one"
- End B: `.abcd/development/brief/04-surfaces/34-build.md:743` — "In the lane's worktree it runs the spec's close (which ships the intent and parks the fidelity receipt)"

05-intent says the impact is demanded at the close that ships the intent: a close with neither a recorded impact nor one given on the call is refused, and planning leaves the judgement owed when none is given. 34-build's landing runs the spec's close in the lane's worktree to ship the intent. Yet none of its start checks (key, ready, open questions, claim sections, hold, blocked, steps, peers) asks for an impact, and the landing passes none to the close. A READY intent planned without an impact therefore builds through every lane and is refused at its last landing. As written, the build cannot finish such an intent.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 407.
