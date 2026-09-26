---
schema_version: 1
id: "iss-2609252243286219"
slug: "capture-reframe-s-whole-write-refuses-with-the-frame-s"
severity: "minor"
category: "ux"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/reframe.go"
resolution: "a whole write with no distinct state inside the fingerprintable history refuses naming that, how many commits the history reaches back and to which commit, and why the state before cannot be compared"
impact: fix
resolved_by:
  commit: "44ad3c43"
---

capture reframe's whole write refuses with 'the frame's previous state cannot be fingerprinted: carries no H2 section titled Construal' when no distinct prior state lies within the history that carries a Construal section, instead of naming the true reason: no prior distinct state within the fingerprintable history, and how far back that history reaches

## Grounds

- pursued: a history whose older states carry no Construal section refuses with 'matches no prior committed state within its fingerprintable history' and the reach; a refusal naming a fingerprint failure would show it wrong
