---
schema_version: 1
id: "iss-2609292234505986"
slug: "invariant-19-in-the-brief-names-the-drain-rule-as-capture"
severity: "nitpick"
category: "drift"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25, lane remedyRequired"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/03-invariants.md"
remedy: "name the rule as eligibility in internal/core/capture/eligible.go, reached through capture.PlanDrain"
---

Invariant 19 in the brief names the drain rule as capture.Eligibility, an exported function that no longer exists: commit 81bfda851 unexported it to eligibility, so the invariant cites a symbol a reader cannot find
