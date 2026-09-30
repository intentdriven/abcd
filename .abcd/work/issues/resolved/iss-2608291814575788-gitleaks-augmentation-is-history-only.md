---
schema_version: 1
id: "iss-2608291814575788"
slug: "gitleaks-augmentation-is-history-only"
severity: "minor"
category: "architectural-insight"
source: "impl-review"
found_during: "ultra-v0.6.8-followup"
found_at: "internal/core/history/history.go"
remedy: "Build the 2026-09-25 technical ruling: the scanner declares an Augmenter interface and a WithAugmenter option, the gitleaks adapter is wired at the composition root, ScanText and ScanBundle append its findings deduplicated on file, line and span, and ErrConfiguredNotFound makes launch fail closed while capture, history and memory write and record the gap. Prove it with a fake augmenter whose finding every consumer (capture, memory ingest, launch dry-run, repolint, the CLI scan) must report, and a not-found case per consumer."
resolution: "The scanner declares an Augmenter seam and WithAugmenter; cmd/abcd wires the gitleaks adapter as the default every scanner.New picks up, so capture, history, memory, launch, the privacy lint, disembark pack and the other write paths report what an armed gitleaks finds. The not-installed binary is a gap: launch and pack fail closed, the privacy lint errors, and the write paths write on the native scanner and name the gap in their receipt."
impact: additive
resolved_by:
  commit: "3d60073dc"
---

ultra-v0.6.8 altitude 2: the opt-in gitleaks augmentation is bolted onto history.Capture alone (internal/core/history/history.go), while the other write paths that build scanner.New — capture, memory ingest, launch dry-run, repolint, the CLI scan — never see it. Deeper fix: fold the gitleaks adapter into internal/adapter/scanner so scanner.New reads the opt-in, ScanText/ScanBundle return the union, and ErrConfiguredNotFound surfaces as the existing Unavailable state; every consumer then inherits it.

## Remedy grounds (2026-09-29)

- Why: the design is ruled in .abcd/work/DECISIONS.md (2026-09-25), so the remedy applies it; it supersedes the body's proposal to fold the adapter into internal/adapter/scanner, which the ruling shows is an import cycle.
- Rejected: keeping the augmentation on history.Capture alone, which leaves every other scanner consumer without the opt-in the repository armed.

## Grounds

- pursued: every scanner consumer reports a fake augmenter's finding and behaves on the not-found gap as the 2026-09-25 ruling says (tests per consumer); a consumer that builds a scanner and neither reports an armed augmenter's finding nor names its gap would show it wrong
