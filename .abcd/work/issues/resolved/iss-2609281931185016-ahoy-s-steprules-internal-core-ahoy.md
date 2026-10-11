---
schema_version: 1
id: "iss-2609281931185016"
slug: "ahoy-s-steprules-internal-core-ahoy"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-lmw127"
origin: researcher-authored
production_mode: hand-written
resolution: "stepRules creates the rules.json skeleton through fsutil.CreateExclusiveIn (createRepoJSON); an existing file is kept and not reported as written"
impact: fix
resolved_by:
  commit: "10b9a31cf"
---

ahoy's stepRules (internal/core/ahoy/apply.go stepRules) writes the empty .abcd/rules.json skeleton through writeRepoJSON (an atomic rename, no re-check) after the interactive prompt phase, so a rules.json written after detection set rules.missing (detect.go) is replaced by the empty-domains skeleton and the receipt says it wrote rules: a silent loss of a hand-written override, with the whole prompt phase as the window.

## Grounds

- pursued: a rules.json written after detection survives ahoy's apply byte for byte and the receipt carries no rules write; TestRulesSkeletonNeverReplacesARulesFileWrittenMeanwhile going red again, or a rules write reported for a kept file, would show it wrong
