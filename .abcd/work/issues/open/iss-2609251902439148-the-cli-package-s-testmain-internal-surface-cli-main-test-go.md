---
schema_version: 1
id: "iss-2609251902439148"
slug: "the-cli-package-s-testmain-internal-surface-cli-main-test-go"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/main_test.go"
---

The cli package's TestMain (internal/surface/cli/main_test.go) re-enters the abcd binary whenever ABCD_CLI_TEST_AS_BINARY=1 is in the environment, whatever the arguments, so an ambient marker makes go test of the package run zero tests, print the bare status board and pass. Test-only; the shipped binary carries no such symbol.
