---
schema_version: 1
id: "iss-2610262030388384"
slug: "dependencies-names-repoprompt-and-codex"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 26 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/02-dependencies.md:34"
remedy: "none (filed automatically)"
finding_ref: cfk-a63dad9c3cb77366
---

Dependencies names RepoPrompt and codex as oracle adapters; ahoy says host-delegated is the only oracle answer with an adapter

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/02-dependencies.md:34` — "RepoPrompt and codex are two such adapters; a **direct API oracle is a legitimate adapter** too."
- End B: `.abcd/development/brief/04-surfaces/01-ahoy.md:893` — "The oracle backend is not asked while host-delegated is the only answer with an adapter"

The dependencies page says RepoPrompt and codex are two of the opt-in oracle adapters behind the seam. The ahoy chapter says the oracle backend question is not asked because host-delegated is the only answer with an adapter, and every other answer is recorded without changing how reviews run. Both cannot describe which oracle adapters exist.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 26.
