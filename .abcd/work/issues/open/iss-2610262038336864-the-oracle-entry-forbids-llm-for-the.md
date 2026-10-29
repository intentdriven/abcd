---
schema_version: 1
id: "iss-2610262038336864"
slug: "the-oracle-entry-forbids-llm-for-the"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 803 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/oracle.md:6"
remedy: "In the oracle row of the seams table in 02-adapters.md change 'host-delegated LLM' to 'host-delegated oracle' (and 'a model' in the interface cell to the oracle's wording if desired)."
finding_ref: cfk-1a1f664cf6083c66
---

The oracle entry forbids "LLM" for the oracle; the adapters chapter names the oracle seam's native default "host-delegated LLM".

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/oracle.md:6` — "forbidden_synonyms: ["LLM", "model", "bot", "AI", "chatbot"]"
- End B: `.abcd/development/brief/05-internals/02-adapters.md:22` — "host-delegated LLM — abcd emits the prompt, the host's subagent dispatch runs it (adr-25)"

oracle.md lists "LLM" among the words that must not stand for the oracle. 02-adapters' seam table names the oracle seam's native default "host-delegated LLM", which is that substitution.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 803.
