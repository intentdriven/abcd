---
schema_version: 1
id: "iss-2610040758571292"
slug: "the-load-rule-domain-governs-one-session"
severity: "minor"
category: "process"
source: "managed-repo"
found_during: "downstream brief-authoring lab report, 2026-10-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/rules/defaults/rules.json"
remedy: "state the cross-session consent protocol in the LOAD domain"
---

The LOAD rule domain governs one session's own load experiments (one owned process group, consent with a cap below the core count on a live machine) and says nothing about load another session on the same machine is about to start. A downstream session asked a peer session go, wait until, or not tonight before loading a 17 GB local model, and reported back once the machine was clean; the exchange worked, and the protocol is stated nowhere.
