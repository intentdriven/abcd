---
schema_version: 1
id: "iss-2610031012543135"
slug: "this-repository-s-agents-md-is-37-109"
severity: "minor"
category: "tech-debt"
source: "review-followup"
found_during: "AGENTS.md draft review, 2026-10-03"
origin: researcher-authored
production_mode: hand-written
found_at: "AGENTS.md"
remedy: "Move material from AGENTS.md under the on-demand rules loader (itd-3's design; the Attribution and Concurrent sessions sections overlap the COMMITTING and DOGFOODING domains) until AGENTS.md is sized below 32 KiB; keep the canary word of itd-2610030814013772's acceptance check in AGENTS.md's first section, so the canary measures the one-file rule and not the cap."
resolution: "AGENTS.md is 26,595 bytes, inside the 32,768-byte reader cap: the Concurrent sessions and Attribution sections keep the short form of each rule and name the repo domains CONCURRENCY and ATTRIBUTION in .abcd/rules.json, which carry the whole text and are injected on recall; the canary line stays in the first section."
impact: internal
resolved_by:
  commit: "7b54dbcf4171dfeab8816cf7f61e5f0122457a10"
---

This repository's AGENTS.md is 37,109 bytes, past Codex's 32 KiB (32,768-byte) default instruction limit, so a Codex session stops reading before the end of the file and works without the conventions that follow the cut. The one-conventions-file rule (itd-2610030814013772, adr-2610030814023326) changes a count of files, not bytes: a file past a native reader's cap is, for that reader, not one file.

## Grounds

- pursued: AGENTS.md stays below 32 KiB and the moved detail is recalled by the loader (TestAgentsMDFitsTheSmallestReadersCap, TestAgentsMDDetailMovedUnderTheLoaderIsRecalled); a host reading only AGENTS.md acting against a rule whose short form dropped a load-bearing clause would show it wrong.
