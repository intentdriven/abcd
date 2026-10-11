---
schema_version: 1
id: "iss-2608261133204171"
slug: "scanner-config-ancestor-guard-covers-one"
severity: "nitpick"
category: "security"
source: "agent-finding"
found_during: "bughunt-round-8"
found_at: "internal/adapter/scanner/scanner.go:126"
resolution: "the config read goes through os.[redacted-user] and fsutil.ReadGuardedIn[redacted-user], refusing an escape at either ancestor; the ancestor test covers both"
impact: fix
resolved_by:
  commit: "a867aa35"
---

scanner.New lstats only the .abcd ancestor while its config lives two levels down, so a symlinked .abcd/config walks the pii.json read out of the tree; the read belongs in the ReadGuardedInRoot form