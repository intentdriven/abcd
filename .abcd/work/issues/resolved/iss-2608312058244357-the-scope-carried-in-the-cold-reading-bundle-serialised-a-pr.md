---
schema_version: 1
id: "iss-2608312058244357"
slug: "the-scope-carried-in-the-cold-reading-bundle-serialised-a-pr"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
resolution: "The bundle now carries a pathless projection of the scope: the kinds and record ids it was scoped to, plus a count of location narrowings and never the locations. The manifest keeps the full resolved scope, because the auditor's account may name paths and the reading's may not."
impact: fix
resolved_by:
  intent: "itd-199"
  spec: "spc-69"
---

The scope carried in the cold-reading bundle serialised a preset's path selectors verbatim, so a repository path reached the reading's own working set and breached brief invariant 15's pathless-bundle rule; the manifest may carry paths and the bundle may not, and one Scope type was being written to both

## Grounds

- pursued: one Scope type written into two artefacts with different disclosure rules is a leak waiting for the first preset that names a path, and brief invariant 15 admits no exception. What would show this wrong is a reading that cannot do its job without knowing WHERE its subset came from, which would mean the pathless bundle and the scope operand are in genuine conflict.
