---
schema_version: 1
id: "iss-2610230814124362"
slug: "a-gitutil-stored-line-test-counts-zero"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "lane C preflight of feat/local-model-pin-r2, 2026-10-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gitutil/stored_each_test.go"
remedy: "Find why the counted git run is not observed under load (a shared cache served the first read, or the count is read before the call completes) and make the count deterministic: isolate the cache per test and read the count after the read returns"
---

TestTheStoredLineReadEndsWithItsContext (internal/gitutil/stored_each_test.go:56) failed in a make preflight of feat/local-model-pin-r2 on 2026-10-23 with 'git ran 0 times; want once, for the first read only', at a load of roughly 27 to 40 on 16 cores, and passes on origin/main and on that branch when run alone. The test counts git invocations, so a run that never reaches git, or whose count is read before git is invoked, under load reports zero.
