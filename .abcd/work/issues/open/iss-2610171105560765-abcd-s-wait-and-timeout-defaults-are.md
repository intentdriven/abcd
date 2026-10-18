---
schema_version: 1
id: "iss-2610171105560765"
slug: "abcd-s-wait-and-timeout-defaults-are"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "product thinker, 2026-10-17, after two preflight timeouts"
origin: researcher-authored
production_mode: hand-written
found_at: "Makefile"
remedy: "First check the state of the art with primary sources (Bazel's per-size test timeouts, CI job timeouts such as GitHub Actions' 360-minute default, timeouts derived from a run's own history). Then derive each default from the project's own measured history, for example a multiple of the recent 95th-percentile queue wait or stage time recorded by past runs, with a floor at today's values, overridable in .abcd/config.json, and reported in the refusal so a person sees which limit ended the wait and how it was chosen."
---

abcd's wait and timeout defaults are fixed numbers, while what they wait for grows with the project. The load gate gives up after 30 minutes by default and after 2 hours in preflight, and go test kills a package at 10 minutes unless told otherwise. On 2026-10-17 two preflights here lost hours to these limits without any check failing: one waited the full 2 hours at the gate and gave up, and another was killed when one package took about ten minutes on a loaded machine. As a managed project gets more packages, tests and concurrent sessions, every fixed default is outgrown in turn, and each time a person or session must notice it and raise it by hand.
