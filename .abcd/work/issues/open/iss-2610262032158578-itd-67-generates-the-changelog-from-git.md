---
schema_version: 1
id: "iss-2610262032158578"
slug: "itd-67-generates-the-changelog-from-git"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 242 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-67-installable-versioned-plugin.md:48"
remedy: "none (filed automatically)"
related_intents: [itd-67]
finding_ref: cfk-8ef8b69f405815ff
---

itd-67 generates the changelog from git tags, commit/spec history and the launch report; the scope page composes it from the records that reached a terminal folder

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-67-installable-versioned-plugin.md:48` — "generated from the canonical history (git tags + commit/spec history + launch report) — never hand-curated prose"
- End B: `.abcd/development/brief/01-product/04-scope.md:42` — "the checkpoint is the derived release, whose changelog is composed from the records that reached a terminal folder"

itd-67 defines an auto-recorded changelog generated from the canonical history (git tags, commit/spec history and the launch report). 04-scope says the checkpoint is the derived release, whose changelog is composed from the records that reached a terminal folder. The two give incompatible sources for the changelog.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 242.
