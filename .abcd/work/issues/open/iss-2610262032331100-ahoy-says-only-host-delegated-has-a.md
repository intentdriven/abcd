---
schema_version: 1
id: "iss-2610262032331100"
slug: "ahoy-says-only-host-delegated-has-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 286 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/01-ahoy.md:893"
remedy: "In 01-ahoy.md (the oracle backend paragraph near line 993) say the oracle.backend question has only host-delegated wired, and that reviewer routing is set elsewhere: oracle.review (host or rp, for build lane reviews), oracle.roles and the provider blocks; stop implying no second reviewer has an adapter."
finding_ref: cfk-289e83a7f56d74f0
---

Ahoy says only host-delegated has a reviewer adapter; build routes lane reviews to RepoPrompt

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/01-ahoy.md:893` — "The oracle backend is not asked while host-delegated is the only answer with an adapter"
- End B: `.abcd/development/brief/04-surfaces/34-build.md:542` — "A build lane's ruthless and security reviews can go to RepoPrompt over MCP instead of the host's own agents"

01-ahoy's install does not ask for the oracle backend, because host-delegated is the only answer with an adapter, and it says other reviewers arrive later. 34-build's reviewer route sends a lane's ruthless and security reviews to RepoPrompt over MCP through oracle.review. So either a second reviewer already has an adapter, which contradicts ahoy, or two different settings choose the reviewer and neither document says how they relate.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 286.
