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
resolution: "The marker write and the embark dry run both ask access(2) first whether the file's folder lets abcd create a file there, so the probe predicts the write's skip, with the same note, for a target root that cannot take a new file."
impact: fix
resolved_by:
  commit: "bc2e92abbf72997dffc29d88849a577150103851"
---

The embark probe mispredicts the marker for an unwritable target root: with the root at mode 0555, a writable .abcd/ and no AGENTS.md, probe reports the marker as install while from skips it with permission denied, because the dry run (ahoy.EnsureMarker) classifies an absent file as missing without asking whether the folder can take a new file, so the comment on embarkMarker that probe 'cannot mispredict the file or the action' holds only for the state of the file itself.

## Grounds

- pursued: TestEmbarkProbePredictsAnUnwritableRoot, a mode-0555 root with a writable .abcd/, absent and current AGENTS.md, asserts probe and from give the identical skip and the records land; a differing probe marker would show it wrong
