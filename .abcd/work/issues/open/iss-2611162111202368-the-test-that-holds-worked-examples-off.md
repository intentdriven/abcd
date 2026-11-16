---
schema_version: 1
id: "iss-2611162111202368"
slug: "the-test-that-holds-worked-examples-off"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "security review of the worked examples for every visible command"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/examples_test.go"
remedy: "read the set of consent flags from the guard registry's person-only acts, or a short list beside the test, and refuse any example passing one unless the command's usage line requires it"
---

The test that holds worked examples off a flag answering a confirmation in advance catches --yes only, so a future example passing another such flag, for instance an --approve the person alone may give, would pass it
