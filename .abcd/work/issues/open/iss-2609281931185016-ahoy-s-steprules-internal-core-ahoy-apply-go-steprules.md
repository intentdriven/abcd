---
schema_version: 1
id: "iss-2609281931185016"
slug: "ahoy-s-steprules-internal-core-ahoy-apply-go-steprules"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-lmw127"
origin: researcher-authored
production_mode: hand-written
---

ahoy's stepRules (internal/core/ahoy/apply.go stepRules) writes the empty .abcd/rules.json skeleton through writeRepoJSON (an atomic rename, no re-check) after the interactive prompt phase, so a rules.json written after detection set rules.missing (detect.go) is replaced by the empty-domains skeleton and the receipt says it wrote rules: a silent loss of a hand-written override, with the whole prompt phase as the window.
