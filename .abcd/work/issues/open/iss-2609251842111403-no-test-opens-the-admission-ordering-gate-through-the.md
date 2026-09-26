---
schema_version: 1
id: "iss-2609251842111403"
slug: "no-test-opens-the-admission-ordering-gate-through-the"
severity: "minor"
category: "tech-debt"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "evals/coldreading_rehearsal_test.go"
---

No test opens the admission ordering gate through the comparative channel's real writer: the rehearsal hand-plants the marker (evals/coldreading_rehearsal_test.go:1397), and the comparative eval ingests but never dispositions, so a writer that stops populating candidate_run would leave every such run un-answerable with every gate green (review-admission 3).
