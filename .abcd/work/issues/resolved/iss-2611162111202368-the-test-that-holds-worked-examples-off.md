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
resolution: "The worked-example test now refuses an example passing any flag that gives a person's yes (--yes, --confirm, --proceed, --replace-existing, --adopt, --approve, --accept-config-change, --flip or --take) unless the command's usage line requires that flag. It found the example for implement resume passing --take, which now names the run with --run."
impact: internal
resolved_by:
  commit: "e36ae5636"
---

The test that holds worked examples off a flag answering a confirmation in advance catches --yes only, so a future example passing another such flag, for instance an --approve the person alone may give, would pass it
