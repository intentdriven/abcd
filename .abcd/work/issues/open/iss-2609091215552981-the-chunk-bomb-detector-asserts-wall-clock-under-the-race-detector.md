---
schema_version: 1
id: "iss-2609091215552981"
slug: "the-chunk-bomb-detector-asserts-wall-clock-under-the-race-detector"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "adversarial-review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/container_test.go"
---

TestPNGChunkCountBombIsBounded proves the payload scan refuses a decompression bomb, and it proves it by timing the run: it measures elapsed wall clock and fails past ten seconds. Under the race detector the same run takes fifteen times as long, so the test reports the chunk loop as unbounded at about twenty-three seconds where it takes about one and a half seconds plain, and the failure message asserts the opposite of what the result it prints shows, because the very same result carries the refusal the bound produced, naming more than the permitted number of compressed chunks. The lane this breaks is not an optional one: make preflight runs the race-enabled internal tests and CI runs them on macOS and Linux both, so the branch that added the test cannot pass its own gate on any machine, and the redness says the security control failed when the control worked. A wall-clock assertion also cannot distinguish a slow machine from a broken bound, which is the distinction the test exists to make. Fix: assert the bound rather than the duration, since the scan already reports why it refused, and keep a duration assertion only if it is scaled for the race detector or skipped under it, so the test measures the property it names. Detector: with the chunk bound removed the test must fail on the missing refusal, and with the bound in place it must pass both plain and under -race on a machine where the plain run takes one second and the instrumented run takes twenty.
