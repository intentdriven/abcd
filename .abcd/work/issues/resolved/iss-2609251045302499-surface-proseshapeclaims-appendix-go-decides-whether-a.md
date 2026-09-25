---
schema_version: 1
id: "iss-2609251045302499"
slug: "surface-proseshapeclaims-appendix-go-decides-whether-a"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
resolution: "ProseShapeClaims marks every byte of a line either mdrecord rule reads as fenced as code, so a tilde-fenced sub-verb invocation is a claim as a backtick-fenced one is."
impact: fix
resolved_by:
  commit: "ac0f03ee"
---

surface ProseShapeClaims (appendix.go) decides whether a sub-verb path is written as an invocation through codeRegions, which pairs backtick runs and so covers a backtick fence but never a tilde fence. A sub-verb path inside a tilde-fenced example, written without an abcd prefix, is not a claim, so a stale sub-verb there passes the brief surface drift check silently. Probe: prose '~~~ / capture list / ~~~' against a tree registering 'abcd capture list' yields no claim. Found in the fence sweep beside TestNoSecondFenceRule, which does not flag it because codeRegions keeps no per-line state.

## Grounds

- pursued: the capture's probe now yields the claim on its line (TestProseShapeClaimsReadATildeFenceAsCode) and the brief appendix tests over the committed chapters still pass; a missing claim for a tilde-fenced path, or a new false drift on a committed chapter, would show it wrong
