---
schema_version: 1
id: "iss-2608211432389181"
slug: "readverdictfile-lstat-then-readfile"
severity: "nitpick"
category: "bug"
source: "user-observation"
found_during: "bughunt-b/round-5"
found_at: "internal/core/intent/audit.go"
resolution: "Read the verdict operand through fsutil.ReadGuarded, closing the lstat-then-read window"
impact: internal
resolved_by:
  commit: "888189b"
---

readVerdictFile does Lstat-then-os.ReadFile with a comment claiming the swap window is not a symlink escape, but os.ReadFile follows a swapped-in symlink and ignores the size cap; it is the last CLI-operand reader not routed through fsutil.ReadGuarded