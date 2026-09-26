---
schema_version: 1
id: "iss-2609261312108500"
slug: "ahoy-connect-s-provider-block-write-re-reads-abcd-config"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25 (integration lane integ2)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/oracle/connect.go"
---

ahoy connect's provider-block write re-reads ~/.abcd/config.json under its lock with plain json.Unmarshal (internal/core/oracle/connect.go writeProviderBlockLocked), so a config.json that names a key twice, or two spellings of one, is collapsed last-wins and rewritten without the other spelling. LoadAPI refuses such a file through layered's jsonstrict walk before the call, but the rewrite judges the bytes it re-reads under the lock, and those it reads with the silent decoder: a file edited between the check and the lock is laundered. The sibling of iss-2609260120380520 on the credential store.
