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
---

abcd drain's --max help says the default is all, but a terminal drive's cap is eight lanes, so a person reading only the help for a long run would not raise it and the run would end after eight.
