---
schema_version: 1
id: "iss-2611040554308925"
slug: "the-lab-chapter-s-exit-codes-say-exit-1"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "the v0.16.1 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/31-lab.md"
remedy: "Confine 'with the artefact rendered and the finding recorded' to the preflight and the sweep, and say the other two exit 1 cases write nothing."
---

The lab chapter's exit codes say exit 1 covers a gate that refused (the preflight, the sweep, the harvest's citation check) or a halted lab refusing a probe, 'with the artefact rendered and the finding recorded' (.abcd/development/brief/04-surfaces/31-lab.md:175-176). That holds for the preflight and the sweep only. The harvest's citation refusal writes nothing, as lines 169-171 of the same chapter say, and sets no halt; `lab record` on a halted lab writes nothing and names the standing finding.
