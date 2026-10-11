---
schema_version: 1
id: "iss-2608210934566226"
slug: "itd-132-adr-persistence-verification"
severity: "major"
category: "security"
source: "user-observation"
found_during: "itd-132 adversarial reviews 2026-08-21"
resolution: "dispositioned by adr-0046 (persistence never weakens the verification posture); the fetch-time and at-rest invariants are carried into hooks/bootstrap.sh by the spc-35/c9cf9e19 series"
impact: internal
---

ADR to draft at itd-132 planning (SPLIT-routed trust rule): binary persistence must not weaken the spc-21 verification posture. Whatever shape itd-132 takes (relocate-execution or download-cache+copy), the fetch-time invariants carry over verbatim — same-origin checksums.txt, HTTPS pin with -q-first curl, proxy/CA env unset, no environment seam for origins — AND the at-rest window is addressed: under the cache model a tampered binary survived at most one update cycle; a persistent location extends that window unboundedly unless the recorded binary_sha256 is re-verified at adoption/refresh points. Companion brief invariant: SessionEnd performs no network work. Route: ADR + brief invariant, filed with the itd-132 spec.