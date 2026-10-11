---
schema_version: 1
id: "iss-2608211432384430"
slug: "history-rootsha-error-message-omits-64"
severity: "nitpick"
category: "bug"
source: "user-observation"
found_during: "bughunt-b/round-5"
found_at: "internal/core/history/history.go"
resolution: "Name both 40- and 64-char widths in the rootSHA diagnostic via a shared const beside the regex"
impact: fix
resolved_by:
  commit: "ecbc899"
---

history Capture/List reject an invalid rootSHA with a message naming only 40 chars though rootSHARe accepts 40 or 64 (SHA-256); the diagnostic contradicts the widened regex