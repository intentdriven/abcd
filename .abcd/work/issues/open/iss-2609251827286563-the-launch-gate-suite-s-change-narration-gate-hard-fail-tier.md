---
schema_version: 1
id: "iss-2609251827286563"
slug: "the-launch-gate-suite-s-change-narration-gate-hard-fail-tier"
severity: "major"
category: "bug"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/launch/gates.go"
---

The launch gate suite's change-narration gate (hard-fail tier) refuses present-state prose: its constructs in internal/core/launch/gates.go flag 'The token used to authenticate the request is read from the environment.' (a participle 'used to'), 'Files that are no longer present in the tree are skipped.' (a state, not a change of abcd), 'The output is renamed to match the tag.' (a present operation), and 'Now, as previously noted, the report lists every gate.' (previously and now with no change verb, which itd-65 AC10 says must not hard-fail; the AC10 test covers the two words only in separate sentences). The finding text never names the docs-lint escape. A doc sentence that narrates nothing refuses the cut, and on a hand-pushed tag the release workflow's archive verify refuses after the tag exists. Remedy: 'used to' only as the past-habit construction, 'no longer' and 'renamed' only beside a change subject naming abcd or its behaviour, a change verb required beside previously/now, and the escape named in the finding.
