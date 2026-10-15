---
id: dcn-2607120000000001
slug: clean-slate-hardening-run-step-0-triage
date: 2026-07-12
---
2026-07-12 — clean-slate hardening run STEP 0 triage. A fresh adversarial
sweep (15 ruthless + 9 security reviewers over current `main`, every finding
independently verified) returned 34 real findings (19 CONFIRMED, 15 PLAUSIBLE,
0 REJECTED; full corpus `.abcd/.work.local/logs/clean-slate-run/sweep-findings.json`).
Key result: the sweep INDEPENDENTLY RE-CONFIRMED the 2026-07-08 review's
code-defect backlog (iss-29/30/32/33/34) is real and still unfixed — prior runs
deferred those code fixes for docs-reconciliation (iss-35/36) and itd-80 feature
work. Draining them is this run. Two BLOCKs found: the scanner serialises a
finding's snippet masking only its own token, leaking sibling secrets on the
same line (iss-65). Triage disposition: newer-package findings (scanner, rules,
intent, spec, frontmatter, lint receipt-gate, capture concurrency, core) minted
as iss-65..72; older-package findings map to existing homes — memory ingest
(C12/C13/P11)→iss-30, atomic-write/fsutil (P1/P6/seed2)→iss-32, ahoy install
(C2/C3)→iss-33, launch glob panic (C11)→iss-34, identity fail-open (C8/P12)→iss-63,
history redaction (C6/C7/P2)→iss-29. iss-70's C16 fix adds `policy.detector` to
the receipt-JSON schema — a record-lint CONTRACT change flagged for maintainer
sign-off before landing (a STOP-adjacent design surface). Ledger triage committed
to `main` as a `chore:` record commit (matches prior record-to-main practice;
keeps the fix branches clean); each code fix lands on its own `auto/*` branch + PR.
