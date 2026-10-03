---
schema_version: 1
id: "iss-2610032202263648"
slug: "the-embark-probe-mispredicts-the-marker-for-an-unwritable"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25 (lane agentsStep5, from step 3's reviews)"
origin: researcher-authored
production_mode: hand-written
remedy: "Predict the refusal in the dry run without writing: when the target file is absent, ask the kernel whether its folder can take a new entry (syscall.Access with W_OK on the parent, in the standard library on darwin and linux, so no new dependency), and return the refusal the write would give; grounds: probe and from share one path so the probe can be trusted, and access(2) answers for the real uid without creating anything."
---

The embark probe mispredicts the marker for an unwritable target root: with the root at mode 0555, a writable .abcd/ and no AGENTS.md, probe reports the marker as install while from skips it with permission denied, because the dry run (ahoy.EnsureMarker) classifies an absent file as missing without asking whether the folder can take a new file, so the comment on embarkMarker that probe 'cannot mispredict the file or the action' holds only for the state of the file itself.
