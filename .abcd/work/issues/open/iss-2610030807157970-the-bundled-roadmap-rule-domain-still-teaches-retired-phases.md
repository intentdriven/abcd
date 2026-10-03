---
schema_version: 1
id: "iss-2610030807157970"
slug: "the-bundled-roadmap-rule-domain-still-teaches-retired-phases"
severity: "minor"
category: "drift"
source: "managed-repo"
found_during: "peer question from a sibling session's abcd user test, answered by run A's orchestrator, 2026-10-03"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/rules/defaults/rules.json"
remedy: "Replace the bundled ROADMAP rules in internal/core/rules/defaults/rules.json with the statement adr-2609212115255771 decision 8 prescribes (the text the repo override at .abcd/rules.json:127-131 already carries), keep or narrow the recall words so a prompt about phases still meets the retirement, add a test that no bundled default rule teaches a phase or a milestone as current, then drop the repo override once it repeats the default; grounds: the record itself (decision 8), no outside practice involved"
---

The bundled ROADMAP rule domain still teaches retired phases and milestones. internal/core/rules/defaults/rules.json:34 injects 'The roadmap is intent-driven; each phase ends in a milestone.' into every managed repository on a prompt that names a roadmap, milestone or phase, although adr-2609212115255771 retired the phase, the milestone and the word roadmap (decision 8 replaces this domain with the decision's statement) and adr-2609292012006845 carries that forward. abcd's own repository masks the defect with a repo override (.abcd/rules.json:127-131), so its own sessions never see it; a sibling session's user test of an adopted repository on v0.9.0 reported it on 2026-10-03. The brief-wide sweep iss-2609251618079479 does not name the defaults file.
