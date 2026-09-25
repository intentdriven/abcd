---
schema_version: 1
id: "iss-2609251827104081"
slug: "the-launch-gate-suite-s-hook-compliance-row-fails-open-on-a"
severity: "major"
category: "bug"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/launch/gates.go"
---

The launch gate suite's hook-compliance row fails open on a hooks config that does not parse: hookComplianceGate (internal/core/launch/gates.go) skips an unparseable hooks file with a comment claiming the smoke refuses it, and the smoke does not, because hookCommandEntries (installsurface.go) also drops an unparseable hooks document silently and the light tier checks existence only. A plugin shipping a hooks.json the host cannot parse registers no hooks on any install, while the row reads 'ran, 0 concern(s)'. itd-65 AC5 promises the concern is surfaced. Remedy: a hooks config that is present and cannot be read or parsed is a finding of the row, and the resolver refuses it as it refuses an unparseable plugin manifest.
