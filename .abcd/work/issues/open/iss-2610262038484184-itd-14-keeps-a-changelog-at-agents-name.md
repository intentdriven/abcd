---
schema_version: 1
id: "iss-2610262038484184"
slug: "itd-14-keeps-a-changelog-at-agents-name"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 819 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-14-prompt-registry-versioning.md:30"
remedy: "none (filed automatically)"
related_intents: [itd-14]
finding_ref: cfk-ef029a806fca1870
---

itd-14 keeps a changelog at agents/< name>/CHANGELOG.md; 01-agents keeps the changelog out of agents/, at .abcd/development/agents/

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-14-prompt-registry-versioning.md:30` — "Per-prompt changelog (`agents/<name>/CHANGELOG.md`)"
- End B: `.abcd/development/brief/05-internals/01-agents.md:71` — "the roster's README and changelog, plain docs with no agent frontmatter, live at [`.abcd/development/agents/`] (../../agents/README.md) instead (iss-110)"

itd-14 puts one changelog per prompt at agents/< name>/CHANGELOG.md. 01-agents says the roster's changelog lives at .abcd/development/agents/, because the loader registers what sits in agents/ and that directory holds only prompts and their fixture directories. itd-14 puts the changelog where the agent-directory contract excludes it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 819.
