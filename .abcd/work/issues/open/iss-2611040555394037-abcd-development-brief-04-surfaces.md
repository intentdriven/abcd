---
schema_version: 1
id: "iss-2611040555394037"
slug: "abcd-development-brief-04-surfaces"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/09-reflect.md"
remedy: "Say which figures the text render shows and which only --json carries."
---

.abcd/development/brief/04-surfaces/09-reflect.md:53-54 and 71-72 say `abcd reflect <release-tag>` renders each intent's honoured, diverged and missing gap counts, plus verdict and gap distributions and the dates of this tag and the previous one. The text render (internal/surface/cli/reflect.go:341, 368) shows the acceptance rollup per intent and a metrics line of shipped, audited, unaudited and this tag's date. The --json form carries the gaps, the rollup and, where there is a previous tag, its date (internal/core/reflect/seed.go:102-105).
