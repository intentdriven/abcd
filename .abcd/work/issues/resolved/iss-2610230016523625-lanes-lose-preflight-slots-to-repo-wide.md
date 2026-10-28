---
schema_version: 1
id: "iss-2610230016523625"
slug: "lanes-lose-preflight-slots-to-repo-wide"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "lane C, 2026-10-22 crash recovery"
origin: researcher-authored
production_mode: hand-written
found_at: "Makefile"
remedy: "Add a make precheck target that runs preflight's fast gates and the repository-wide audit tests (the whole-tree prose, reference, surface and lint tests) without the full unit and race stages, needing no load gate, and have AGENTS.md's build section tell a lane to run it before queuing a preflight"
resolution: "make precheck runs preflight's gates, build, vet, the record, page and agent audit packages, and the command and site audit tests chosen by name, in four to six minutes with no queue, and AGENTS.md tells a lane to run it before queuing a preflight; tests hold its gates to preflight's and its test lines to what it claims"
impact: internal
resolved_by:
  commit: "808bb1574e6b4bba393f4e8f168a88862c48c71b"
---

Lanes lose whole preflight slots to repo-wide audit tests that fail late. A preflight queues behind every other preflight on the machine (an hour or more on 2026-10-22), then spends its first long stretch on the unit stage before a test that reads the whole tree fails on a sentence: TestSurfaceChapterProseStatesNoShape failed two lanes' preflights on 2026-10-22, each for a command name in the build brief's prose, and the lane then had to queue again from the back. The same lesson came up on 2026-10-18, when lanes were tested package by package and missed the repository-wide audit tests until preflight. Nothing lets a lane run the fast, repository-wide gates and audit tests on their own, in seconds or a few minutes, before it takes a preflight slot.
