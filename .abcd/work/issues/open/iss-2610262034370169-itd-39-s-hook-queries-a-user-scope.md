---
schema_version: 1
id: "iss-2610262034370169"
slug: "itd-39-s-hook-queries-a-user-scope"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 510 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-39-scope-aware-memory-retrieval.md:35"
remedy: "none (filed automatically)"
related_intents: [itd-39]
finding_ref: cfk-32e2baa51075efde
---

itd-39's hook queries a user-scope memory store at ~/.abcd.noindex/memory/; the library chapter lists user-scope memory outside the repo as not built.

A sequencing impossibility (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-39-scope-aware-memory-retrieval.md:35` — "the hook queries the `index.md` of both memory scopes (`<workspace>/.abcd/memory/` and `~/.abcd.noindex/memory/`)"
- End B: `.abcd/development/brief/04-surfaces/07-library.md:415` — "user-scope memory outside the repo"

itd-39 (planned) has the prompt hook query the index of both memory scopes, the workspace .abcd/memory/ and ~/.abcd.noindex/memory/, with the narrower scope winning, and its decisions say release placement no longer waits because itd-36's store has shipped. 07-library's account of what ships lists user-scope memory outside the repo under Not built, and the store it documents lives in the repo at .abcd/memory/ alone. Nothing in itd-39's scope builds the second scope, so its two-scope query depends on a store no record delivers.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 510.
