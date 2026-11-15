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
resolution: "The lab chapter's exit codes now give 'the artefact rendered and the finding recorded' to the preflight and the sweep only, and say the harvest's citation check writes nothing and sets no halt and a halted lab refusing a probe writes nothing and names the standing finding. The detector is a script in the lane directory, not a committed test: the widen to a test file was refused at the reservation push (no push credential in the session; the task needs the sandbox lifted or a credential for a committed test)."
impact: fix
resolved_by:
  commit: "0dedbdbbeb88ba41b0d0837306f35c9d9888eaef"
---

The lab chapter's exit codes say exit 1 covers a gate that refused (the preflight, the sweep, the harvest's citation check) or a halted lab refusing a probe, 'with the artefact rendered and the finding recorded' (.abcd/development/brief/04-surfaces/31-lab.md:175-176). That holds for the preflight and the sweep only. The harvest's citation refusal writes nothing, as lines 169-171 of the same chapter say, and sets no halt; `lab record` on a halted lab writes nothing and names the standing finding.

## Grounds

- pursued: the exit 1 clause of .abcd/development/brief/04-surfaces/31-lab.md matches internal/core/lab (harvest.go's citation refusal and record.go's halted refusal write nothing; preflight.go and sweep.go write the artefact and a gate finding); a refused harvest or a refused probe on a halted lab that leaves a file or a new finding in the lab home would show it wrong
