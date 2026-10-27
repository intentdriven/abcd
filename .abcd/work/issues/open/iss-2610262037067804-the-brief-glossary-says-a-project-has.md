---
schema_version: 1
id: "iss-2610262037067804"
slug: "the-brief-glossary-says-a-project-has"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 698 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/brief.md:28"
remedy: "none (filed automatically)"
finding_ref: cfk-fb02a2b297b98e4e
---

The brief glossary says a project has exactly one brief; the build chapter gives every lane a brief of its own.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/brief.md:28` — "A project has exactly one brief."
- End B: `.abcd/development/brief/04-surfaces/34-build.md:617` — "The brief is rendered from the lane's base commit, read out of git's objects rather than the worktree's files"

brief.md keeps "brief" for the project-level document under `.abcd/development/brief/` and says a project has exactly one. 34-build.md uses "the brief", unqualified, for the per-lane document rendered for the implementer from the lane's base commit, and names a lane stage after it. The word carries a second sense the brief entry does not allow.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 698.
