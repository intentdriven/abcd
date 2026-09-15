---
schema_version: 1
id: "iss-2608311234541392"
slug: "commands-reading-md-tells-a-host-that-every-list-level-refus"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
resolution: "The page states that a refusal records only once the run's identity is proven, and the front door renders the result on a recorded refusal so refusal_record is reachable through it."
impact: fix
resolved_by:
  intent: "itd-185"
  spec: "spc-63"
---

commands/reading.md tells a host that every list-level refusal leaves a refusal record, but four of the six list-level refusals it enumerates (wrong _type, unresolvable run, manifest hash mismatch, drifted definition) deliberately write nothing, because the run's identity is not yet proven. The same page instructs the host to report refusal_record from the JSON, which the front door can never emit: a refusal returns an exitError and the render is never reached, so IngestResult.RefusalPath is dead surface through the only front door.

## Grounds

- pursued: the page describes what the verb does, and a surface test proves the rendered refusal_record names a file on disk; a host following the page and finding nothing would show this wrong
