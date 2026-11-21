---
schema_version: 1
id: "iss-2610262031246229"
slug: "the-press-release-calls-the-review-model"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 125 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/01-press-release.md:27"
remedy: "Reword 'a host-delegated LLM with opt-in oracle adapters' (line 27) to 'a host-delegated oracle with opt-in oracle adapters (adr-25)'; the same phrase stands in 05-internals/02-adapters.md:22 and 05-internals/04-universal-patterns.md:175 and wants the same edit."
finding_ref: cfk-d4fb06f64b604c1c
resolution: "The press release's plumbing line now names 'a host-delegated oracle with opt-in oracle adapters (adr-25)' instead of the forbidden synonym 'host-delegated LLM', and a lint test holds it there. The drive's guard refused the remedy's same edit at 05-internals/02-adapters.md:22 (which iss-2610262038336864 tracks) and 05-internals/04-universal-patterns.md:180, so both are still to do."
impact: fix
resolved_by:
  commit: "3fbfe5669309b47a37772fe195c67dc72b972a14"
---

The press release calls the review model a host-delegated LLM; the oracle glossary entry forbids LLM as a synonym for oracle.

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/01-press-release.md:27` — "a host-delegated LLM with opt-in oracle adapters (native, CLI, API, MCP)"
- End B: `.abcd/development/brief/glossary/core/oracle.md:6` — "forbidden_synonyms: ["LLM", "model", "bot", "AI", "chatbot"]"

The press release's plumbing line lists 'a host-delegated LLM with opt-in oracle adapters (native, CLI, API, MCP)', which is the thing the glossary calls the host-delegated oracle. The oracle entry declares 'forbidden_synonyms: ["LLM", "model", "bot", "AI", "chatbot"]', because the generic word elides the review-and-decide role. The brief's lead document substitutes a forbidden synonym for the canonical term.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 125.

## Grounds

- pursued: the press release names the oracle seam's default by the glossary term; TestPressReleaseNamesTheHostDelegatedOracle fails if 'host-delegated LLM' returns to the line or the adr-25 wording goes.
