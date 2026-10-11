---
schema_version: 1
id: "iss-2608210934566227"
slug: "itd-132-principle-durable-state-platform"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "itd-132 adversarial reviews 2026-08-21"
resolution: "principle landed as adr-0046: persistence never weakens the verification posture"
impact: internal
---

Principle to draft at itd-132 planning (SPLIT-routed stance): store durable state where the platform documents it survives — never fight the host lifecycle inside a directory the host re-clones or garbage-collects. Surfaced by the 2026-08-21 plugin-update post-mortem (binary in the re-cloned cache dir). Candidate home: .abcd/development/principles/. File alongside the itd-132 spec.