---
schema_version: 1
id: "iss-2610262032437661"
slug: "ahoy-says-the-user-scope-abcd-noindex"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 307 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/01-ahoy.md:470"
remedy: "none (filed automatically)"
finding_ref: cfk-52daa7c00e2bfbc2
---

ahoy says the user-scope ~/.abcd.noindex/memory/ resolves nothing today; 07-memory has the prompt hook querying the index of both the repo and the user memory scope

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/01-ahoy.md:470` — "curated cross-project memory pages (a later phase; nothing resolves it today)"
- End B: `.abcd/development/brief/05-internals/07-memory.md:193` — "The hook queries the `index.md` of both memory scopes (repo / user — § 0)."

01-ahoy's inventory of the user-scope home marks memory/ as curated cross-project pages of a later phase that nothing resolves, and puts the person's own recalled notes in memory.json beside the project's .abcd/memory.json. 07-memory states as current that personal preferences land in ~/.abcd.noindex/memory/ and that the hook queries the index.md of both the repo and the user scope. Both cannot describe the same machine: either the hook reads a user-scope memory/ index or nothing resolves that directory.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 307.
