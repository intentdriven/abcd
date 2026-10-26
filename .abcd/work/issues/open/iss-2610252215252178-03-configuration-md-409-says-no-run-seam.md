---
schema_version: 1
id: "iss-2610252215252178"
slug: "03-configuration-md-409-says-no-run-seam"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "brief single-source study, 2026-10-25, reader S13"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md"
remedy: "Say that owed audits drain through intent audit --owed and the build lane's validate stage, and keep only the run-seam drain at iteration boundaries as staged."
---

03-configuration.md:409 says 'No run seam ships, so nothing drains today', but commands/intent.md:1028 documents intent audit --owed draining the owed reviews, and 01-agents.md:153 says the same
