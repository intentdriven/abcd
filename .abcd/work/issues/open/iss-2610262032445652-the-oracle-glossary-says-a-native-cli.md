---
schema_version: 1
id: "iss-2610262032445652"
slug: "the-oracle-glossary-says-a-native-cli"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 311 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/01-ahoy.md:898"
remedy: "none (filed automatically)"
finding_ref: cfk-e3e3d4e386b2fe3b
---

the oracle glossary says a native, CLI, API or MCP oracle adapter is wired behind the seam; ahoy says every backend answer but host-delegated changes nothing about how reviews run

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/01-ahoy.md:898` — "every answer but host-delegated is recorded without changing how reviews run"
- End B: `.abcd/development/brief/glossary/core/oracle.md:28` — "a concrete **oracle adapter** — native, CLI, API, or MCP — is wired behind the same seam."

core/oracle defines the alternative to host delegation as a concrete oracle adapter, native, CLI, API or MCP, wired behind the same seam. 01-ahoy says the oracle backend is not asked because host-delegated is the only answer with an adapter, and that every other answer is recorded without changing how reviews run. A reader cannot both wire such an adapter and find that choosing one does nothing.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 311.
