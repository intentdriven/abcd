---
schema_version: 1
id: "iss-2611061656052853"
slug: "the-implement-loop-package-takes-2189-s"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "macOS gate readings of two release cuts"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/ci.yml"
refines: [iss-2611061059489399]
remedy: "Read more runs to find why the package ran slower on its own leg, cut the real-time waits of its slowest tests through the speed lane, or split its three test processes across two legs, since another leg finishes in about 7 minutes."
---

The implement loop package takes 2189 s and then 2137 s on its own macOS leg against a 2400 s limit, slower than the 1173 s and 1504 s it took on a shared leg. The cause is not yet found, and runners vary about twofold, so the own leg itself is not shown to be the reason. Its ten slowest tests spend about 600 s in real-time waits. Every drive test added to the package eats the same margin, and a slow runner fails the release gate with no failing test.
