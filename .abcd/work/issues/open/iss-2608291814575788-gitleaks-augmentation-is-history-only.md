---
schema_version: 1
id: "iss-2608291814575788"
slug: "gitleaks-augmentation-is-history-only"
severity: "minor"
category: "architectural-insight"
source: "impl-review"
found_during: "ultra-v0.6.8-followup"
found_at: "internal/core/history/history.go"
deferred_after: "v0.10.0"
deferral_reason: "The orchestrator's technical ruling of 2026-09-25 in .abcd/work/DECISIONS.md settles the design (an Augmenter interface the scanner declares, gitleaks wired at the composition root, ErrConfiguredNotFound failing launch closed and recorded as a gap by capture, history and memory); it is built in its own lane, not in the scanner-cluster fix round that recorded the ruling."
---

ultra-v0.6.8 altitude 2: the opt-in gitleaks augmentation is bolted onto history.Capture alone (internal/core/history/history.go), while the other write paths that build scanner.New — capture, memory ingest, launch dry-run, repolint, the CLI scan — never see it. Deeper fix: fold the gitleaks adapter into internal/adapter/scanner so scanner.New reads the opt-in, ScanText/ScanBundle return the union, and ErrConfiguredNotFound surfaces as the existing Unavailable state; every consumer then inherits it.
