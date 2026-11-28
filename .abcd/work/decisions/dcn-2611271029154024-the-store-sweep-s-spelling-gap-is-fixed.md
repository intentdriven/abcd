---
id: dcn-2611271029154024
slug: the-store-sweep-s-spelling-gap-is-fixed
date: 2026-11-27
---
2026-11-27 — The store sweep's spelling gap is fixed, superseding dcn-2611270841518400. scanner.SweepCallerHome, scanner.SurvivingCallerHome and the home_path_self detector now read the caller's home in every spelling fsutil.HomeSpellings gives RedactHome (as written, cleaned, resolved through symlinks, the macOS data-volume path, and both Unicode normal forms), through the same escape and percent-encoding views as before, so the memory, history and ideate stores, the refusal path and the privacy lint's own-home check agree with RedactHome on those spellings. To reverse it, have the three read the home as written only (drop the callerHomeSpellings expansion in internal/adapter/scanner) and restore the earlier decision; the per-store tests named in iss-2609290055092630 will then fail.
