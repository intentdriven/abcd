---
schema_version: 1
id: "iss-2608261437043634"
slug: "the-live-disciplines-bucket-was-never"
severity: "minor"
category: "observation"
source: "agent-observation"
found_during: "bughunt-b-round-9"
found_at: ".abcd/development/intents/disciplines"
resolution: "disciplines swept onto intent-auditor and intent audit; spc-28 records the late sweep"
impact: fix
resolved_by:
  commit: "eff3168c"
---

the live disciplines bucket was never swept for the spc-28 intent-auditor rename