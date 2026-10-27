---
schema_version: 1
id: "iss-2610262038210994"
slug: "the-internals-index-says-the-config"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 789 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/README.md:9"
remedy: "none (filed automatically)"
finding_ref: cfk-926e7c21303f6a0a
---

The internals index says the config chapter covers seam-backend and adapter-registry config; the config chapter says there is no adapters registry block and the backend selectors are staged.

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/README.md:9` — "`config.json` schema, including its `meta` setup block (seam-backend + adapter-registry config)"
- End B: `.abcd/development/brief/05-internals/03-configuration.md:404` — "`adapters` registry block, because the wired-adapter registry it indexed has no"

05-internals/README says 03-configuration holds the `config.json` schema including seam-backend and adapter-registry configuration. 03-configuration says there is no `adapters` registry block because the registry it indexed has no home, and lists the seam backend selectors as staged keys that nothing reads.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 789.
