---
schema_version: 1
id: "iss-2609011424033603"
slug: "the-brief-s-surface-prose-has-drifted-from-the-shipped-binar"
severity: "major"
category: "drift"
source: "agent-finding"
found_during: "the v0.7.0 cut, iss35 full-tier crosscheck"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/"
resolution: "all twenty-one pre-existing prose findings are corrected against the code: the required --grounds on promote and resolve, the seven-check ready gate and the Grounds section in the intent template, the ahoy, disembark, launch, site and identity details, the guard citation, and the index rows and bare-status enumerations, which the chapters point at rather than restate"
impact: fix
resolved_by:
  commit: "b18c7c5d8eaf7e1153d106f5f4432b0595b25641"
---

the brief's surface prose has drifted from the shipped binary in twenty-one places the machine-checked layer cannot see

## Grounds

- pursued: prose the surface_coverage rule cannot scan is brought to what the binary does, and duplicated enumerations are collapsed to one home so they cannot drift twice; a re-run of the iss35 crosscheck at this commit reporting any of the twenty-one findings would show it wrong
