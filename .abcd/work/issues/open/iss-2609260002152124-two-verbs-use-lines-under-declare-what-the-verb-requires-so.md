---
schema_version: 1
id: "iss-2609260002152124"
slug: "two-verbs-use-lines-under-declare-what-the-verb-requires-so"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/history.go, internal/surface/cli/cli.go (capture disposition)"
---

Two verbs' Use lines under-declare what the verb requires, so the usage line, the worked example check and the aggregated refusal all read a requirement set smaller than the one the verb enforces. history discard refuses without --yes on every call while its Use line is discard <staged-filename>; capture disposition requires --grounds on every state except held, which requires --exit-condition instead, while its Use line brackets both as optional. Confirmed at 6e4eca9a while writing the worked examples for iss-2609100508565741: a disposition example without --grounds and a discard example without --yes are both refused.
