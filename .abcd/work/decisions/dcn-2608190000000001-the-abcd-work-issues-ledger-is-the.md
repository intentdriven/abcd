---
id: dcn-2608190000000001
slug: the-abcd-work-issues-ledger-is-the
date: 2026-08-19
---
2026-08-19 — The `.abcd/work/issues/` ledger is the single canonical issue store; forge
issues (GitHub) are a derived, opt-in mirror surface — consistent with the host-delegated
boundary (forges are hosts). Per-field ownership: the ledger owns existence, content, and
resolution; the forge owns nothing. Sync is one-way mirror-out, the forge id written back
into the record as the echo suppressor; forge-side activity (triage labels, a close) is
never auto-synced — it generates an import proposal a human accepts via an explicit import
that writes the canonical file with provenance. A forge-side close never imported is
reopened by the next mirror pass: the mirror is self-healing by construction. Autonomous
hunts file findings as fingerprint-keyed iss-N records inside the gated PR they already
open, validated by record-lint (the armed detector) — never as forge issues. Grounded in
commissioned SOTA research (see
`../development/research/notes/2026-08-19-issue-ledger-forge-sync-sota.md`): one-way
canonical sync is the surviving pattern (coreos/issue-sync post-mortem); agent findings as
records-via-reviewed-PR is 2025-26 practice (GitHub advisory-database, ClusterFuzz
fingerprint dedup); ungated agent filing degrades the whole record (curl, 2025).
Architecture-shaping: graduate to an ADR.
