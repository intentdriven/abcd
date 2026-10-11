---
schema_version: 1
id: "iss-2608211146222734"
slug: "roadmap-intents-banned-literal-outside"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "bughunt-b-round-4"
found_at: "AGENTS.md"
resolution: "Replaced 'roadmap/intents' with 'roadmap, intents' in both the AGENTS.md and .abcd/README.md durable-record maps, killing the banned pre-adr-30 literal while keeping both real siblings and the one intentional nested path (decisions/adrs)."
impact: internal
---

The retired pre-adr-30 path literal 'roadmap/intents' — a record-lint blocker token (moved-intents-path: 'intents live at intents/') — survived in AGENTS.md and .abcd/README.md, the two canonical where-things-live maps, both outside record-lint's roots (.abcd/development) so the armed token is structurally blind to them. Read in context beside the real nested path decisions/adrs, the slash reads as a path separator, i.e. the banned pre-adr-30 layout. A 2026-07-06 review already adjudicated both sites stale (fix #6). CLAUDE.md is a symlink to AGENTS.md, so the line is injected into every agent session and risks an agent propagating the banned path into development/ where it is a live blocker. Live instance of open iss-279's detector gap (banned_tokens cannot reach AGENTS.md/CONTRIBUTING.md).