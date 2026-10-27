---
schema_version: 1
id: "iss-2610262032354048"
slug: "ahoy-calls-the-remote-apply-the-one-abcd"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 290 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/01-ahoy.md:90"
remedy: "none (filed automatically)"
finding_ref: cfk-1c4698c18919957c
---

ahoy calls the remote apply the one abcd verb that mutates state outside the machine; land pushes branches to origin and queues pull requests

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/01-ahoy.md:90` — "the one abcd verb that mutates state outside this"
- End B: `.abcd/development/brief/04-surfaces/38-land.md:4` — "pushes the branch to `origin`, and queues its open pull request where the merge gate allows"

The ahoy chapter says the remote apply is the only abcd verb that changes anything outside this machine. The land chapter describes a shipped verb that pushes the branch to origin and arms and queues its pull request on the forge. Both are writes outside the machine, so the remote apply cannot be the only one.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 290.
