---
schema_version: 1
id: "iss-2610262040093616"
slug: "the-adapters-chapter-says-the-repoprompt"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 913 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/02-adapters.md:239"
remedy: "none (filed automatically)"
related_intents: [itd-6]
finding_ref: cfk-b952a84a494552b2
---

The adapters chapter says the RepoPrompt route opens a new chat for each reviewer; itd-6 requires re-reviews to stay in the same RP chat, never a new one

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/02-adapters.md:239` — "with `new_chat: true` when the tool declares it, so each reviewer is fresh."
- End B: `.abcd/development/intents/shipped/itd-6-rp-mcp-only-integration.md:45` — "never `--new-chat`, never fresh `rp builder`"

02-adapters says the RepoPrompt adapter runs one MCP session per review and sends `new_chat: true` when the tool declares it, so each reviewer starts fresh. itd-6 says a re-review after fixes must stay in the same RP chat, never `--new-chat`, and carries the chat_id from one call to the next. A route that always opens a fresh chat cannot keep a re-review in the same chat.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 913.
