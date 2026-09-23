---
schema_version: 1
id: "iss-2609231947544298"
slug: "the-load-check-s-stray-rule-lifetime-cpu-share-of-at-least-0"
severity: "minor"
category: "inconsistency"
source: "impl-review"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/machineload/classify.go"
---

The load check's stray rule (lifetime CPU share of at least 0.9, spc-2609231542463113 section 3) cannot flag the 2026-09-22/23 incident it is claimed to flag: at the incident's load of about 440 on 16 cores no busy loop can hold 0.9 of a core, so another account's loops read as no strays at all (other_strays count 0) and the warning comes from the extreme trigger alone. itd-2609231434459890's Mechanism says the foreign-sustained-CPU signal flags both incidents; under oversubscription it flags only the first. TestMechanismFlagsIncidentTwo pins both halves (a 0.95-share fixture is counted; the oversubscribed shape warns through the extreme trigger only). Fixing it needs a ruling on the stray definition (a share relative to what the machine could give, or a second sample), which the lane cannot take.
