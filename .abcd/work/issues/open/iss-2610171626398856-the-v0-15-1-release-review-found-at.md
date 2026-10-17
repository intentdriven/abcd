---
schema_version: 1
id: "iss-2610171626398856"
slug: "the-v0-15-1-release-review-found-at"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.1 release review"
origin: researcher-authored
production_mode: hand-written
found_at: "CHANGELOG.md"
remedy: "Announce abcd status and abcd announce in the release that ships itd-2609151838312703, and resolve the two-preflights records iss-2610161639025702 and iss-2610161756384596 that the release review found still open."
---

The v0.15.1 release review found at CHANGELOG.md:19-72: INCOMPLETE (judgement call). The 0.15.1 section has no Added line for user-facing surfaces that are new since v0.15.0 and present in this binary. (1) `abcd status`, a top-level verb in `abcd --help` with a user-invocable commands/status.md (block: people); commands/status.md does not exist at v0.15.0. (2) `abcd announce` (commands/announce.md). Both come from the planned itd-2609151838312703 and its open spc-2610150728396006. (3) `abcd implement load --race-gate` and the `load.preflights` / ABCD_LOAD_PREFLIGHTS setting (internal/core/implement/load_slot…
