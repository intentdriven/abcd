---
schema_version: 1
id: "iss-2608312019544150"
slug: "prefixdenied-in-the-reading-assembler-compares-case-sensitiv"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
resolution: "prefixDenied now folds case, like segmentDenied beside it, so deny.go's header comment that the deny binds every path component case-insensitively is true of both halves. itd-199 gave the lexical claim a second consumer in validPresetPath, whose stated contract is that a path which looks like it reaches a denied place is refused at the door."
impact: fix
resolved_by:
  intent: "itd-199"
  spec: "spc-69"
---

prefixDenied in the reading assembler compares case-sensitively while segmentDenied folds case, and deny.go's own header comment claims the deny binds every path component case-insensitively, so the comment is wrong about the boundary it documents

## Grounds

- pursued: the asymmetry admitted nothing while the walk was the only consumer, but a second consumer now rests on the lexical claim alone, and on a case-insensitive filesystem that claim was all that held. What would show this wrong is a denied prefix whose case genuinely distinguishes two different paths, which would make folding it the error instead.
