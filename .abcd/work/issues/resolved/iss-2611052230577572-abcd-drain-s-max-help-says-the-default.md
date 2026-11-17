---
schema_version: 1
id: "iss-2611052230577572"
slug: "abcd-drain-s-max-help-says-the-default"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "release cut preparation"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/drain.go"
remedy: "Say in the flag's help that a drive's default cap is eight lanes."
resolution: "The drain help now says a drive caps at eight lanes when no cap is named, as the code does."
impact: fix
resolved_by:
  commit: "b456024c6a1abac75c26e20d4d9e28e903bc2154"
---

abcd drain's --max help says the default is all, but a terminal drive's cap is eight lanes, so a person reading only the help for a long run would not raise it and the run would end after eight.
