---
schema_version: 1
id: "iss-2609261447331434"
slug: "ahoy-offers-trufflehog-as-an-optional-dependency-the-deps"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/detect.go"
resolution: "ahoy no longer emits the deps.trufflehog_missing gap: nothing in abcd runs trufflehog, so the explain-then-install mode would have offered to install a program that does nothing for the person. A test pins that no dependency gap names trufflehog."
impact: fix
resolved_by:
  commit: "920afe82"
---

ahoy offers trufflehog as an optional dependency (the deps.trufflehog_missing gap, 'trufflehog enables deep secret scanning when scan.deep=true', fix hint 'brew install trufflehog') although nothing in abcd runs trufflehog: no adapter exists and scan.deep is read by no scanner. The gap asks a person to install a tool abcd never uses, which the explain-then-install mode (itd-63) would turn into an executed install of a program that does nothing for them.

## Grounds

- pursued: we expect no ahoy run to offer trufflehog while no adapter runs it; shown wrong if a trufflehog gap or install offer reappears without a wired trufflehog adapter
