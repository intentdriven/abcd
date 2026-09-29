---
schema_version: 1
id: "iss-2608291814575788"
slug: "gitleaks-augmentation-is-history-only"
severity: "minor"
category: "architectural-insight"
source: "impl-review"
found_during: "ultra-v0.6.8-followup"
found_at: "internal/core/history/history.go"
deferred_after: "v0.11.1"
deferral_reason: "a lane of its own: the design is settled by the 2026-09-25 technical ruling in .abcd/work/DECISIONS.md (an Augmenter interface the scanner declares, gitleaks wired at the composition root, ErrConfiguredNotFound failing launch closed and recorded as a gap by capture, history and memory), and the build, which touches every scanner consumer, is not done at v0.11.1."
---

ultra-v0.6.8 altitude 2: the opt-in gitleaks augmentation is bolted onto history.Capture alone (internal/core/history/history.go), while the other write paths that build scanner.New — capture, memory ingest, launch dry-run, repolint, the CLI scan — never see it. Deeper fix: fold the gitleaks adapter into internal/adapter/scanner so scanner.New reads the opt-in, ScanText/ScanBundle return the union, and ErrConfiguredNotFound surfaces as the existing Unavailable state; every consumer then inherits it.
