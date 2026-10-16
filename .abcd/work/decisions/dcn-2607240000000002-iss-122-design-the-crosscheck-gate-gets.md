---
id: dcn-2607240000000002
slug: iss-122-design-the-crosscheck-gate-gets
date: 2026-07-24
---
2026-07-24 — iss-122 design: the crosscheck gate gets a committed input
manifest (doc list, directions, checker count, prompt hash) with TIERED
depth — full depth for feature/breaking releases, Direction-B-only shallow
pass for patch. The receipt must echo the manifest hash AND the tier;
receipt_gate refuses a receipt whose tier mismatches the release's declared
impact. Automated refusal is PROCEDURAL ONLY (manifest/tier mismatch,
undispositioned findings); confirmed findings go to the maintainer, whose
PROMOTE with recorded dispositions is the gate (verifier-selects-gates-
decide). Rejected: hard-blocking on confirmed majors (a stochastic LLM
triage could block a release); never-worse ratchet (drift persists; tiers
make counts incomparable).
