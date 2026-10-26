---
schema_version: 1
id: "iss-2610252211476480"
slug: "two-intent-sentences-name-less-than-the"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "docs-currency review of docs/, 2026-10-24, reader D6"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/surface/sentences.go"
remedy: "Reword both write clauses within SentenceCap to name the provider route's ingest (and, for audit, the drift report), for example 'Emit the consistency request over the brief and every intent, or one intent: Writes the request, or on a provider route the review; refuses a superseded intent.' (160 characters), then regenerate surface.json and the CLI reference."
---

Two intent sentences name less than the verb writes. intent audit (internal/core/surface/sentences.go:242-243; page line 3061) says 'Writes an OWED stub only for --owed or an id', but --issue-drift leaves a report under .abcd/.work.local/logs/audit/issue-drift-<ts>/ (internal/core/capture/drift.go:68-73,179), and on a provider route audit <itd-N> sends the request and ingests the verdict into the shipped intent's Audit Notes (internal/surface/cli/cli.go:3446-3448, dispatch.go:179-211). intent consistency (sentences.go:248-249; page line 3123) says 'Writes the request locally', but on a provider route it sends each request and ingests the answers, writing the dated review and one capture per finding (internal/surface/cli/intent_consistency.go:105-121). Both sentences predate provider dispatch (51c7a9744). Same class as iss-2610251559136898, which holds five other sentences.
