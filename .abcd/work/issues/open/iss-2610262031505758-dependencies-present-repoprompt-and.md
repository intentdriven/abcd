---
schema_version: 1
id: "iss-2610262031505758"
slug: "dependencies-present-repoprompt-and"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 185 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/02-dependencies.md:34"
remedy: "none (filed automatically)"
related_intents: [itd-2609170822093401]
finding_ref: cfk-076dad4b95b50fa4
---

Dependencies present RepoPrompt and codex as oracle adapters an operator can select, but itd-2609170822093401 says nothing reads the oracle backend value today.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/02-dependencies.md:34` — "RepoPrompt and codex are two such adapters"
- End B: `.abcd/development/intents/planned/itd-2609170822093401-the-oracle-choice-is-one-repo-wide-value.md:26` — "backend is one repo-wide value that nothing reads"

02-dependencies says concrete oracle backends are opt-in adapters, selected when an operator wants abcd to reach a model directly, and names RepoPrompt and codex as two of them, without marking them staged. itd-2609170822093401 says the oracle backend is one repo-wide value that nothing reads, that every delegated step reaches whatever model the harness picks, and that the provider adapters belong to another intent. The two cannot both describe the present state.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 185.
