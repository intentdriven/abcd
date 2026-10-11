---
schema_version: 1
id: "iss-166"
slug: "ahoy-install-yes-silently-skips-the"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "second-repo install session"
found_at: "internal/core/ahoy/apply.go"
resolution: "the flag excludes the optional identity pin by ruling; the exclusion is stated in its help, the install envelope and the completion output, with the way to apply it"
impact: fix
---

ahoy install --yes silently skips the optional identity-pin category: the flag is documented as approving every resolvable change category without prompting, and it reports 'already up to date', yet the interactive path still offers the optional pin (.abcd/config/identity.json). Either --yes covers optional categories too, or its help and the completion output must say optional categories are excluded and how to apply them.