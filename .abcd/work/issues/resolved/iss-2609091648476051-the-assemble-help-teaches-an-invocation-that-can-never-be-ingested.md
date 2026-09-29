---
schema_version: 1
id: "iss-2609091648476051"
slug: "the-assemble-help-teaches-an-invocation-that-can-never-be-ingested"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "release-gate"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/reading.md"
resolution: "reading assemble reports ingestable and renders an ingest: line for a run written outside the run directory; the help example and the plugin page assemble into the default run directory and name --out as an inspection copy"
impact: fix
resolved_by:
  commit: "400625adc0d0c908b96a23a6c86fb97c7b3ed5e0"
---

The reading assemble verb accepts an output directory, and both its terminal help and its plugin page use one in the worked example, writing the run somewhere other than the default run directory. A run assembled that way can never be ingested: the ingest verb resolves a run's manifest only from the default run directory under its run id, so a manifest parked anywhere else is invisible to it and the run cannot be proven. Nothing says so at assembly time, and nothing says so in the example. The shipped help is therefore teaching the one invocation that leads to a dead end, and it teaches it in the place a reader is most likely to copy from. The cost lands late: the assembly succeeds, the reading is commissioned and returned, and the failure appears at the ingest, by which point the run's output exists and has nowhere to go. Fix direction: either make the example use the default location and document the flag as a dry-run and inspection aid rather than a run location, or teach the ingest to resolve a manifest from a named directory so the flag means what the example implies. Detector: an assembly written outside the default run directory either reports at assembly time that it cannot be ingested, or is ingestable, and the worked example in the help and on the plugin page is one a reader can follow to a committed run.

## Grounds

- pursued: an assembly written with --out now says at assembly time it cannot be ingested, and the worked examples reach a committed run; a named-directory run reading ingestable, or an example still using --out, would show it wrong
