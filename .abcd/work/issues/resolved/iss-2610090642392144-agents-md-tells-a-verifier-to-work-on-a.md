---
schema_version: 1
id: "iss-2610090642392144"
slug: "agents-md-tells-a-verifier-to-work-on-a"
severity: "minor"
category: "inconsistency"
source: "review-followup"
found_during: "2026-10-07/08 autonomous drain and v0.13.3 cut"
origin: researcher-authored
production_mode: hand-written
found_at: "AGENTS.md"
remedy: "Say in AGENTS.md and the CONCURRENCY rule that a verifier's copy goes outside every working tree (the session scratchpad or a machine-store directory), or have the verbs refuse when the nearest checkout above a directory is not the one the directory was extracted from."
resolution: "AGENTS.md and the CONCURRENCY rule now say a verifier's copy goes outside every working tree, in the session's scratchpad or under ~/.abcd.noindex/, never the local tier's scratch/, and a test holds both forms to it."
impact: internal
resolved_by:
  commit: "d875a06ebc2408a41033c9b233155a5b9fc42b2b"
---

AGENTS.md tells a verifier to work on a copy made with git archive into a scratch directory, and the local scratch tier lives inside the worktree. A copy extracted there is not a git checkout, so every abcd verb run from it that reads git or the record store resolves upward to the live worktree that contains it. In the v0.13.3 cut the binary verbs run from the export read the release worktree, which was only equivalent because that worktree was clean at the content commit.

## Grounds

- pursued: a verifier following either form extracts its copy where no enclosing checkout exists, so git -C <copy> rev-parse --show-toplevel fails; a form that drops the location, or a copy that still resolves to a worktree, shows it wrong
