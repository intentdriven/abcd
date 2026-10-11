---
schema_version: 1
id: "iss-2608270908341622"
slug: "ahoy-s-home-redaction-tests-hasprefix-p"
severity: "nitpick"
category: "bug"
source: "agent-finding"
found_during: "issue-sweep-2026-08-27"
found_at: "internal/core/ahoy/fsutil.go"
resolution: "ahoy path/home redaction and fsutil.RedactRoot fold on a case-folding FS, so a case-variant HOME/root is redacted instead of leaking"
impact: fix
resolved_by:
  commit: "aca9d57a"
---

ahoy's home redaction tests HasPrefix(p, home+sep) case-sensitively, so a case-variant HOME spelling leaks the identity path in rendered output on a case-folding filesystem