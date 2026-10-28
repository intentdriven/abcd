---
schema_version: 1
id: "iss-2610262040125215"
slug: "the-configuration-chapter-says-abcd"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 916 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md:339"
remedy: "none (filed automatically)"
related_intents: [itd-6]
finding_ref: cfk-8a1be53cca22e31f
---

The configuration chapter says abcd never configures RepoPrompt or writes its route keys; itd-6 has ahoy find RP's server config and write it into .abcd/config.json

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/03-configuration.md:339` — "abcd never starts, installs or configures RepoPrompt, and never writes either key."
- End B: `.abcd/development/intents/shipped/itd-6-rp-mcp-only-integration.md:43` — "notes it in `.abcd/config.json` → `oracle.rp.mcp_config_path`, and tests reachability"

03-configuration's reviewer-route section says abcd never starts, installs or configures RepoPrompt and never writes either key (`oracle.review`, `oracle.rp`). itd-6 has ahoy find the RP MCP server's configuration, write its path into `.abcd/config.json` as `oracle.rp.mcp_config_path`, test reachability and lock `oracle.backend`. These cannot both be true of abcd's setup.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 916.
