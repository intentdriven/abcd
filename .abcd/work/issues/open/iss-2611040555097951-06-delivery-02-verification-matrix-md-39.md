---
schema_version: 1
id: "iss-2611040555097951"
slug: "06-delivery-02-verification-matrix-md-39"
severity: "major"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/02-verification-matrix.md"
remedy: "State that a re-pack replaces the lifeboat at once and discards delegated output, or make a re-pack over delegated output refuse without an explicit flag."
---

06-delivery/02-verification-matrix.md:39 says a re-run over an abcd-produced destination confirms, then regenerates in place. Nothing confirms: abcd disembark pack has no confirmation flag, internal/core/lifeboat/pack.go has no confirmation step, and a re-run with stdin closed exits 0 at once. The re-run also discards what later steps wrote into the lifeboat: press-release.json, press-release.md and review/ were gone after a re-pack, and neither 04-surfaces/02-disembark.md nor commands/disembark.md warns of it. The row's refusals of a non-empty foreign directory and a malformed provenance file hold.
