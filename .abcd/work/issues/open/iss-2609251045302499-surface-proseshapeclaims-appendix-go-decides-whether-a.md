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
---

surface ProseShapeClaims (appendix.go) decides whether a sub-verb path is written as an invocation through codeRegions, which pairs backtick runs and so covers a backtick fence but never a tilde fence. A sub-verb path inside a tilde-fenced example, written without an abcd prefix, is not a claim, so a stale sub-verb there passes the brief surface drift check silently. Probe: prose '~~~ / capture list / ~~~' against a tree registering 'abcd capture list' yields no claim. Found in the fence sweep beside TestNoSecondFenceRule, which does not flag it because codeRegions keeps no per-line state.
