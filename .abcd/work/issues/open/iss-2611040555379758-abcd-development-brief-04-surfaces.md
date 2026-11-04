---
schema_version: 1
id: "iss-2611040555379758"
slug: "abcd-development-brief-04-surfaces"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reading/parked.go"
remedy: "Make the parked-bundle cap match what assembly can write, or refuse at assembly and name the size and limit."
---

.abcd/development/brief/04-surfaces/23-reading.md:364-381 describes the dispatch form and its refusals with no size limit. ReadParked reads the bundle under reading.MaxFileBytes, 4 MiB (internal/core/reading/parked.go:68; internal/core/reading/assemble.go:298; called first at internal/surface/cli/reading.go:758), and assembly writes bundles with no such cap. On this tree a detection bundle of 7.5 MB and a widening bundle of 7.5 MB exceed it, so `reading ingest --dispatch` exits 2 with a bare file-exceeds-size-cap message before any provider is contacted, and the documented harness-route refusal is not what prints for such a run. An entailment bundle of 2.3 MB is under the cap.
