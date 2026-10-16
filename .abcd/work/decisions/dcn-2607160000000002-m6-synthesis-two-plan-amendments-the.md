---
id: dcn-2607160000000002
slug: m6-synthesis-two-plan-amendments-the
date: 2026-07-16
---
2026-07-16 — M6 synthesis, two plan amendments: the oracle audit is keyed by the
lifeboat's manifest hash (audit/oracle-<manifest12>.json), not the plan's
wall-clock <ts> (no timestamp ever enters a lifeboat artifact); and
_provenance.json is never mutated post-pack — each synthesis artifact
self-records its mode (deterministic|delegated) instead of the plan's
"_provenance records which" (the commit marker stays immutable). Synthesis
outputs (principles, press-release, audit/) join the lessons files outside
manifest_sha256.
