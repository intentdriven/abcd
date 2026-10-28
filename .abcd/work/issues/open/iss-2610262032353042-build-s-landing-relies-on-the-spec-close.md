---
schema_version: 1
id: "iss-2610262032353042"
slug: "build-s-landing-relies-on-the-spec-close"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 289 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/34-build.md:743"
remedy: "none (filed automatically)"
finding_ref: cfk-042ea5abe3c2df26
---

Build's landing relies on the spec close to ship the intent, but no build check secures the impact that launch says the close requires

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/34-build.md:743` — "In the lane's worktree it runs the spec's close (which ships the intent and parks the fidelity receipt)"
- End B: `.abcd/development/brief/04-surfaces/04-launch.md:516` — "the spec verb's close refuses to ship an intent that records none when none is given on the close, and moves nothing"

34-build's landing runs the spec's close in the lane's worktree as the step that ships the intent. None of its start checks (key, ready, open questions, claim sections, hold, blocked, steps, peers) asks for an impact, and the close it runs is given none. 04-launch says the spec verb's close refuses to ship an intent that records no impact when none is given on the close, and moves nothing; it also says a planned intent may still lack an impact. As written, a planned intent with no impact passes every build check and its run then cannot ship at the landing.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 289.
