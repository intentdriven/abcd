---
schema_version: 1
id: "iss-2610262031052928"
slug: "the-press-release-says-disembark-pack"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 85 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/01-press-release.md:23"
remedy: "none (filed automatically)"
finding_ref: cfk-82b72a9489d34b05
---

The press release says `disembark pack` runs three passes, chat retrieval among them, but the agent catalog says the pass A/B/C agents are unbuilt design targets

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/01-press-release.md:23` — "runs three passes (settled artefacts → targeted chat retrieval → distil/compose/audit)"
- End B: `.abcd/development/brief/05-internals/01-agents.md:78` — "The lifeboat pipeline is drawn around a larger roster than the one that ships."

The press release presents `/abcd:disembark pack` as running three passes (settled artefacts, then targeted chat retrieval, then distil/compose/audit), with no staged marker. The agent catalog says the lifeboat pipeline is drawn around a larger roster than the one that ships. It lists the Pass A agents, the Pass B chat-distiller and the Pass C composers as design targets, none of which exists in `agents/`. A pack cannot run a pass whose agents do not exist.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 85.
