---
id: dcn-2607240000000004
slug: itd-28-gitleaks-sign-off-stage-2-runs
date: 2026-07-24
---
2026-07-24 — itd-28 gitleaks sign-off: Stage 2 runs through the scanner
seam — native patterns are the default engine, gitleaks is the stronger
opt-in adapter when the binary is present, and the hook reports which
engine ran (loud staging). CI's gitleaks pass stays the authoritative
backstop; iss-96 pattern parity folds into the spec. No new hard
dependency — adr-22 holds. Rejected: gitleaks as hard local dependency
(first adr-22 exception, per-machine install burden); CI-only Stage 2
(secrets reach pushed remote history before CI sees them).
