---
schema_version: 1
id: "iss-2611051719413677"
slug: "test-reuse-s-classifier-does-not-follow"
severity: "minor"
category: "tech-debt"
source: "agent-observation"
found_during: "build run, 5 November"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/testreuse"
remedy: "Treat a non-literal program name as evidence that the test must run."
---

Test reuse's classifier does not follow a program name held in a local variable, so a test that starts a program that way can replay locally until the hosted run. The classifier is a best-effort read of test source, and the hosted run reuses nothing.
