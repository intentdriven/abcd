---
schema_version: 1
id: "iss-2610121448010531"
slug: "the-dogfood-remote-check-accepts-an-ssh"
severity: "nitpick"
category: "observation"
source: "impl-review"
found_during: "security review round 3 of fix/hooks-run-head-in-dogfood-roots, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/dogfood/dogfood.go"
remedy: "refuse the literal loopback hosts (localhost, 127.0.0.0/8, ::1) in remoteRefusal with a reason naming the residual, and keep the docs' alias caveat"
---

The dogfood remote check accepts an ssh or https remote on this machine: localhost, 127.0.0.0/8, ::1 and [::1]. An agent running as the account can move such a remote, and the docs state it as the person's residual because abcd cannot tell an ssh alias from a forge. Refusing the literal loopback names would cost nothing for a real forge and would match the refusal of filesystem paths, as defence in depth that an alias still escapes.
